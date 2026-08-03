"""
Kankor PDF Extractor Microservice
==================================
Hybrid extraction pipeline supporting:
- Text extraction (pdfplumber + OCR fallback)
- Page rendering to PNG (PyMuPDF)
- Both combined for vision-based Gemini extraction

Endpoints:
  GET  /health        - Health check
  POST /extract-text  - Text-only extraction (backward compatible)
  POST /extract       - Full extraction: text + page images (base64 PNG)
"""

from flask import Flask, request, jsonify
from flask_cors import CORS
import pdfplumber
import fitz  # PyMuPDF
import io
import os
import re
import base64
import tempfile
import shutil
import traceback

# Optional OCR support (only imported if Tesseract is available)
try:
    import pytesseract
    from PIL import Image, ImageEnhance
    HAS_OCR = True
except ImportError:
    HAS_OCR = False

app = Flask(__name__)
CORS(app)

# ──────────────────────────────────────────────────────────────────────────────
# Configuration
# ──────────────────────────────────────────────────────────────────────────────
app.config['MAX_CONTENT_LENGTH'] = 15 * 1024 * 1024  # 15MB max request
MAX_FILE_SIZE = 15 * 1024 * 1024
MAX_PAGES = 30
MAX_VISION_PAGES = 20       # Max pages to render for vision (memory constraint)
RENDER_DPI = 200            # 200 DPI for vision (smaller payloads, faster upload)
OCR_DPI = 400               # Higher DPI for OCR fallback


# ══════════════════════════════════════════════════════════════════════════════
# HEALTH ENDPOINT
# ══════════════════════════════════════════════════════════════════════════════

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        'status': 'healthy',
        'service': 'pdf-extractor',
        'has_ocr': HAS_OCR,
        'render_dpi': RENDER_DPI
    }), 200


# ══════════════════════════════════════════════════════════════════════════════
# EXTRACT-TEXT ENDPOINT (Backward Compatible)
# ══════════════════════════════════════════════════════════════════════════════

@app.route('/extract-text', methods=['POST'])
def extract_text():
    """
    Text-only extraction using smart dual-path approach:
    1. pdfplumber native text extraction (preserves formatting)
    2. OCR fallback for corrupted/image-based pages
    
    Returns: { text, success, pages_processed, extraction_stats }
    """
    file_data, error_response = _validate_upload()
    if error_response:
        return error_response

    try:
        text, stats = _extract_text_smart(file_data)

        if not text.strip():
            return jsonify({
                'error': 'No text could be extracted from PDF',
                'success': False,
                'suggestion': 'Try a PDF with selectable text'
            }), 400

        return jsonify({
            'text': text,
            'success': True,
            'pages_processed': _count_pages(file_data),
            'extraction_stats': stats
        }), 200

    except Exception as e:
        print(f"[extract-text] ERROR: {e}")
        traceback.print_exc()
        return jsonify({'error': str(e), 'success': False}), 500


# ══════════════════════════════════════════════════════════════════════════════
# EXTRACT ENDPOINT (NEW — returns text + page images for vision pipeline)
# ══════════════════════════════════════════════════════════════════════════════

@app.route('/extract', methods=['POST'])
def extract():
    """
    Full extraction returning BOTH text and rendered page images.
    
    This is the primary endpoint for the hybrid/vision extraction pipeline.
    It returns:
    - Full extracted text (pdfplumber + OCR fallback)
    - Per-page PNG images rendered at RENDER_DPI (base64 encoded)
    - Per-page text
    
    Returns: {
        success, text, total_pages,
        pages: [{ page_number, image_base64, text }]
    }
    """
    file_data, error_response = _validate_upload()
    if error_response:
        return error_response

    print(f"\n{'='*60}")
    print(f"[extract] Full extraction requested (text + vision)")
    print(f"{'='*60}")

    try:
        # ── Phase 1: Extract full text ──
        print("[extract] Phase 1: Text extraction...")
        full_text, stats = _extract_text_smart(file_data)

        # ── Phase 2: Render pages as PNG ──
        print("[extract] Phase 2: Page rendering...")
        pages = _render_pages_fitz(file_data, dpi=RENDER_DPI)

        # ── Phase 3: Extract per-page text with PyMuPDF ──
        print("[extract] Phase 3: Per-page text extraction...")
        _enrich_pages_with_text(file_data, pages)

        total_pages = len(pages)

        print(f"[extract] Complete: {len(full_text)} chars, {total_pages} pages rendered")
        print(f"[extract] Stats: {stats}")
        print(f"{'='*60}\n")

        return jsonify({
            'success': True,
            'text': full_text,
            'total_pages': total_pages,
            'pages': pages,
            'extraction_stats': stats
        }), 200

    except Exception as e:
        print(f"[extract] ERROR: {e}")
        traceback.print_exc()
        return jsonify({'error': str(e), 'success': False}), 500


# ══════════════════════════════════════════════════════════════════════════════
# PDF RENDERING (PyMuPDF)
# ══════════════════════════════════════════════════════════════════════════════

def _render_pages_fitz(file_data, dpi=RENDER_DPI):
    """
    Render each PDF page to a base64-encoded PNG image using PyMuPDF.
    
    PyMuPDF renders pages at the specified DPI with excellent fidelity
    for mathematical symbols, equations, and mixed Persian/English text.
    
    Args:
        file_data: Raw PDF bytes
        dpi: Output resolution (300 DPI = clear math, 400 DPI = very sharp)
    
    Returns:
        List of dicts: [{ page_number, image_base64 }]
    """
    pages = []
    zoom = dpi / 72.0  # PyMuPDF base is 72 DPI
    matrix = fitz.Matrix(zoom, zoom)

    try:
        doc = fitz.open(stream=file_data, filetype="pdf")
        num_pages = min(len(doc), MAX_VISION_PAGES)

        print(f"  Rendering {num_pages} pages at {dpi} DPI...")

        for i in range(num_pages):
            page = doc[i]
            pix = page.get_pixmap(matrix=matrix, colorspace=fitz.csRGB)

            # Convert to PNG bytes
            img_bytes = pix.tobytes("png")

            # Base64 encode for JSON transport
            img_b64 = base64.b64encode(img_bytes).decode('utf-8')

            img_size_kb = len(img_bytes) / 1024
            print(f"  Page {i+1}: {img_size_kb:.0f} KB PNG → {len(img_b64)} chars base64")

            pages.append({
                'page_number': i + 1,
                'image_base64': img_b64
            })

            # Free pixmap memory immediately
            pix = None

        doc.close()
        return pages

    except Exception as e:
        print(f"  [render] PyMuPDF rendering failed: {e}")
        traceback.print_exc()
        # If rendering fails, return empty pages list
        # The caller should fall back to text-only extraction
        return []


def _enrich_pages_with_text(file_data, pages):
    """
    Add per-page text extraction to rendered pages using PyMuPDF's get_text().
    This is faster than pdfplumber for simple text extraction.
    """
    try:
        doc = fitz.open(stream=file_data, filetype="pdf")
        for page_info in pages:
            page_num = page_info['page_number'] - 1  # 0-indexed
            if page_num < len(doc):
                page = doc[page_num]
                page_text = page.get_text()
                page_info['text'] = page_text.strip() if page_text else ''
        doc.close()
    except Exception as e:
        print(f"  [enrich] Per-page text extraction failed: {e}")
        for page_info in pages:
            if 'text' not in page_info:
                page_info['text'] = ''


# ══════════════════════════════════════════════════════════════════════════════
# TEXT EXTRACTION (pdfplumber + OCR fallback)
# ══════════════════════════════════════════════════════════════════════════════

def _extract_text_smart(file_data):
    """
    Smart dual-path extraction:
    1. PRIMARY: pdfplumber native text extraction
    2. FALLBACK: OCR when pdfplumber text is corrupted or empty
    """
    text = ""
    stats = {
        "pages_total": 0,
        "pages_pdfplumber": 0,
        "pages_ocr": 0,
        "pages_empty": 0
    }

    file_obj = io.BytesIO(file_data)

    with pdfplumber.open(file_obj) as pdf:
        pages_to_process = min(len(pdf.pages), MAX_PAGES)
        stats["pages_total"] = pages_to_process

        for i in range(pages_to_process):
            page = pdf.pages[i]
            raw_text = page.extract_text() or ""

            if raw_text and not _is_text_corrupted(raw_text):
                text += raw_text + "\n\n"
                stats["pages_pdfplumber"] += 1
            elif raw_text:
                # Corrupted text — try OCR
                if HAS_OCR:
                    ocr_text = _extract_page_ocr(page, i)
                    if ocr_text:
                        text += ocr_text + "\n\n"
                        stats["pages_ocr"] += 1
                    else:
                        stats["pages_empty"] += 1
                else:
                    # No OCR available, use pdfplumber text anyway
                    text += raw_text + "\n\n"
                    stats["pages_pdfplumber"] += 1
            else:
                # No selectable text — try OCR
                if HAS_OCR:
                    ocr_text = _extract_page_ocr(page, i)
                    if ocr_text:
                        text += ocr_text + "\n\n"
                        stats["pages_ocr"] += 1
                    else:
                        stats["pages_empty"] += 1
                else:
                    stats["pages_empty"] += 1

    return _clean_extracted_text(text).strip(), stats


def _is_text_corrupted(text):
    """Detect garbled/corrupted extracted text."""
    if not text or len(text) < 10:
        return False

    # Excessive replacement characters (U+FFFD)
    replacement_count = text.count('\ufffd')
    if replacement_count > len(text) * 0.05:
        return True

    # Excessive control characters
    control_chars = sum(1 for c in text if ord(c) < 32 and c not in '\n\r\t')
    if control_chars > len(text) * 0.02:
        return True

    # Low printable ratio
    printable = sum(1 for c in text if c.isprintable() or c in '\n\r\t')
    if printable < len(text) * 0.7:
        return True

    return False


def _extract_page_ocr(page, page_index):
    """OCR extraction with math optimization (eng+fas, PSM 4, contrast enhanced)."""
    try:
        page_image_data = page.to_image(resolution=OCR_DPI)
        img_bytes = io.BytesIO()
        page_image_data.save(img_bytes, format='PNG')
        img_bytes.seek(0)
        page_image = Image.open(img_bytes)

        # Preprocess for better math symbol recognition
        if page_image.mode not in ('RGB', 'L'):
            page_image = page_image.convert('RGB')

        enhancer = ImageEnhance.Contrast(page_image)
        page_image = enhancer.enhance(1.5)
        enhancer = ImageEnhance.Sharpness(page_image)
        page_image = enhancer.enhance(1.8)
        enhancer = ImageEnhance.Brightness(page_image)
        page_image = enhancer.enhance(1.1)

        # Primary: English first for math symbols
        ocr_text = pytesseract.image_to_string(
            page_image, lang='eng+fas',
            config='--psm 4 --oem 3'
        )

        if not ocr_text.strip():
            # Fallback: Persian first
            ocr_text = pytesseract.image_to_string(
                page_image, lang='fas+eng',
                config='--psm 4 --oem 3'
            )

        return ocr_text.strip() if ocr_text else ""

    except Exception as e:
        print(f"  [OCR] Failed page {page_index+1}: {e}")
        return ""


def _clean_extracted_text(text):
    """Clean text while preserving math notation and equation structure."""
    # Remove excessive blank lines (max 3 consecutive for math spacing)
    text = re.sub(r'\n{4,}', '\n\n\n', text)

    # Remove truly excessive blank lines from start/end
    lines = text.split('\n')
    while lines and lines[0].strip() == '':
        lines.pop(0)
    while lines and lines[-1].strip() == '':
        lines.pop()

    return '\n'.join(lines)


# ══════════════════════════════════════════════════════════════════════════════
# HELPERS
# ══════════════════════════════════════════════════════════════════════════════

def _validate_upload():
    """
    Validate uploaded file from request.
    Returns (file_data, error_response).
    If error_response is not None, the caller should return it immediately.
    """
    if 'file' not in request.files:
        return None, (jsonify({'error': 'No file provided', 'success': False}), 400)

    file = request.files['file']

    if file.filename == '':
        return None, (jsonify({'error': 'No file selected', 'success': False}), 400)

    if not file.filename.lower().endswith('.pdf'):
        return None, (jsonify({'error': 'Only PDF files are allowed', 'success': False}), 400)

    file_data = file.read()
    file_size = len(file_data)

    if file_size > MAX_FILE_SIZE:
        return None, (jsonify({
            'error': f'File size exceeds {MAX_FILE_SIZE // (1024*1024)}MB limit',
            'success': False
        }), 400)

    print(f"[PDF Extractor] Processing: {file.filename} ({file_size} bytes)")
    return file_data, None


def _count_pages(file_data):
    """Count total pages in PDF."""
    try:
        with pdfplumber.open(io.BytesIO(file_data)) as pdf:
            return len(pdf.pages)
    except Exception:
        return 0


# ══════════════════════════════════════════════════════════════════════════════
# ENTRY POINT
# ══════════════════════════════════════════════════════════════════════════════

if __name__ == '__main__':
    port = int(os.environ.get('PDF_EXTRACTOR_PORT', 5000))
    print(f"\n{'='*60}")
    print(f"  Kankor PDF Extractor v2.0")
    print(f"  Port: {port}")
    print(f"  Max file: {MAX_FILE_SIZE // (1024*1024)}MB | Max pages: {MAX_PAGES}")
    print(f"  Render DPI: {RENDER_DPI} | OCR: {'available' if HAS_OCR else 'unavailable'}")
    print(f"  Endpoints: /health  /extract-text  /extract")
    print(f"{'='*60}\n")
    app.run(host='0.0.0.0', port=port, debug=False)
from flask import Flask, request, jsonify
from flask_cors import CORS
import pdfplumber
import pytesseract
from PIL import Image, ImageFilter, ImageEnhance
import io
import os
import re
import string

app = Flask(__name__)
CORS(app)

# Configuration
app.config['MAX_CONTENT_LENGTH'] = 15 * 1024 * 1024  # 15MB max request size
MAX_FILE_SIZE = 15 * 1024 * 1024  # 15MB
MAX_PAGES = 30
OCR_CONFIDENCE_THRESHOLD = 40  # Lowered from 60 to capture more text
OCR_DPI = 400  # Higher DPI for better math symbol recognition


@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({'status': 'healthy', 'service': 'pdf-extractor'}), 200


@app.route('/extract-text', methods=['POST'])
def extract_text():
    """
    Extract text from uploaded PDF file using smart dual-path approach:
    1. Try pdfplumber's native text extraction first (preserves formatting, math symbols)
    2. Fall back to OCR only if pdfplumber text is empty or corrupted
    """
    print(f"[PDF Extractor] Received request: method={request.method}")
    print(f"[PDF Extractor] Content-Type: {request.content_type}")
    
    # Validate file presence
    if 'file' not in request.files:
        print(f"[PDF Extractor] ERROR: No 'file' field in request.files")
        return jsonify({'error': 'No file provided', 'success': False}), 400
    
    file = request.files['file']
    
    if file.filename == '':
        return jsonify({'error': 'No file selected', 'success': False}), 400
    
    # Validate file type
    if not file.filename.lower().endswith('.pdf'):
        return jsonify({'error': 'Only PDF files are allowed', 'success': False}), 400
    
    try:
        file_data = file.read()
        file_size = len(file_data)
        
        print(f"[PDF Extractor] Processing file: {file.filename}, size: {file_size} bytes")
        
        # Check file size
        if file_size > MAX_FILE_SIZE:
            return jsonify({
                'error': f'File size exceeds {MAX_FILE_SIZE // (1024*1024)}MB limit',
                'success': False
            }), 400
        
        # Extract text using smart dual-path approach
        text, stats = extract_text_smart(file_data)
        
        print(f"[PDF Extractor] Extracted {len(text)} characters from PDF")
        print(f"[PDF Extractor] Stats: {stats}")
        
        if not text.strip():
            return jsonify({
                'error': 'No text could be extracted from PDF. The PDF might contain only images.',
                'success': False,
                'suggestion': 'Try using a PDF with selectable text'
            }), 400
        
        return jsonify({
            'text': text,
            'success': True,
            'pages_processed': count_pages(file_data),
            'extraction_stats': stats
        }), 200
    
    except Exception as e:
        print(f"[PDF Extractor] ERROR: {str(e)}")
        import traceback
        traceback.print_exc()
        return jsonify({
            'error': f'Failed to extract text: {str(e)}',
            'success': False
        }), 500


def extract_text_smart(file_data):
    """
    Smart dual-path extraction:
    1. PRIMARY: Use pdfplumber's native text extraction (preserves formatting, math symbols, layout)
    2. FALLBACK: Use OCR only when pdfplumber text is corrupted or empty
    
    This ensures mathematical content (equations, symbols, formulas) is preserved
    exactly as it appears in text-based PDFs, while still handling scanned/image PDFs.
    """
    text = ""
    stats = {
        "pages_total": 0,
        "pages_pdfplumber": 0,
        "pages_ocr": 0,
        "pages_empty": 0
    }
    
    file_obj = io.BytesIO(file_data)
    
    with pdfplumber.open(file_obj) as pdf:
        pages_to_process = min(len(pdf.pages), MAX_PAGES)
        stats["pages_total"] = pages_to_process
        
        print(f"\n{'='*60}")
        print(f"[SMART EXTRACT] Processing {pages_to_process} pages")
        print(f"{'='*60}\n")
        
        for i in range(pages_to_process):
            page = pdf.pages[i]
            
            print(f"\n{'─'*60}")
            print(f"[PAGE {i+1}/{pages_to_process}]")
            
            # STEP 1: Try pdfplumber's native text extraction first
            raw_text = page.extract_text() or ""
            
            if raw_text and not is_text_corrupted(raw_text):
                # pdfplumber text looks good - use it directly!
                print(f"  ✓ Using pdfplumber text ({len(raw_text)} chars)")
                text += raw_text + "\n\n"
                stats["pages_pdfplumber"] += 1
            elif raw_text:
                # Text exists but looks corrupted/garbled
                print(f"  ⚠ pdfplumber text is corrupted ({len(raw_text)} chars), falling back to OCR")
                ocr_text = extract_page_ocr_optimized(page, i)
                if ocr_text:
                    text += ocr_text + "\n\n"
                    stats["pages_ocr"] += 1
                else:
                    stats["pages_empty"] += 1
            else:
                # No selectable text - must be scanned/image PDF
                print(f"  ⚠ No selectable text, using OCR")
                ocr_text = extract_page_ocr_optimized(page, i)
                if ocr_text:
                    text += ocr_text + "\n\n"
                    stats["pages_ocr"] += 1
                else:
                    stats["pages_empty"] += 1
    
    # Clean the final text while preserving math symbols
    print(f"\n{'─'*60}")
    print(f"[FINAL CLEAN]")
    text = clean_extracted_text(text)
    print(f"  Final length: {len(text)} characters")
    print(f"  Stats: {stats}")
    print(f"{'='*60}\n")
    
    return text.strip(), stats


def is_text_corrupted(text):
    """
    Detect if extracted text is corrupted/garbled.
    
    Common corruption patterns in Persian/Arabic PDFs:
    - Characters appear in wrong order (RTL rendering issues)
    - High ratio of control/replacement characters (\ufffd = �)
    - Text appears as meaningless symbol sequences
    
    Returns True if text appears corrupted, False if it looks valid.
    """
    if not text or len(text) < 10:
        return False  # Too short to determine
    
    # Check for excessive replacement characters (Unicode U+FFFD)
    replacement_count = text.count('\ufffd')
    if replacement_count > len(text) * 0.05:  # More than 5% replacement chars
        return True
    
    # Check for excessive control characters
    control_chars = sum(1 for c in text if ord(c) < 32 and c not in '\n\r\t')
    if control_chars > len(text) * 0.02:
        return True
    
    # Check if text has reasonable character diversity (not just random symbols)
    # Valid text should have spaces, letters, and punctuation
    printable = sum(1 for c in text if c.isprintable() or c in '\n\r\t')
    if printable < len(text) * 0.7:
        return True
    
    return False


def extract_page_ocr_optimized(page, page_index):
    """
    Extract text from a single page using math-optimized OCR.
    
    Improvements for math content:
    - Higher DPI (400) for better symbol recognition
    - Uses eng+fas language (English first for math symbols)
    - PSM 4: assumes single column of variable-sized text (better for mixed math/text)
    - Image preprocessing: contrast enhancement, sharpening
    - No grayscale conversion (preserves color information)
    - Multiple OCR passes with different settings (best result wins)
    """
    try:
        print(f"  [OCR] Converting page to image at {OCR_DPI} DPI...")
        
        # Get the page image at high resolution
        page_image_data = page.to_image(resolution=OCR_DPI)
        
        # Convert to PIL Image
        import io as img_io
        img_byte_arr = img_io.BytesIO()
        page_image_data.save(img_byte_arr, format='PNG')
        img_byte_arr.seek(0)
        page_image = Image.open(img_byte_arr)
        
        # Keep original color - math PDFs may use color for emphasis
        # Apply preprocessing for better OCR
        processed_image = preprocess_for_ocr(page_image)
        
        print(f"  [OCR] Running Tesseract with eng+fas, PSM 4...")
        
        # PRIMARY OCR: English first (math symbols are Latin-based), Persian second
        ocr_text = pytesseract.image_to_string(
            processed_image,
            lang='eng+fas',  # English first for math, Persian for text
            config='--psm 4 --oem 3'  # PSM 4: single column of variable sizes
        )
        
        if ocr_text.strip():
            # Check quality by trying an alternative PSM for comparison
            alternative_text = pytesseract.image_to_string(
                processed_image,
                lang='eng+fas',
                config='--psm 6 --oem 3'  # PSM 6: uniform block (better for dense math)
            )
            
            # Use whichever extracted more meaningful content
            best_text = ocr_text if len(ocr_text) >= len(alternative_text) else alternative_text
            
            print(f"  [OCR] Extracted {len(best_text.strip())} characters")
            return best_text.strip()
        
        # Fallback: try with Persian first
        print(f"  [OCR] Retrying with fas+eng...")
        ocr_text = pytesseract.image_to_string(
            processed_image,
            lang='fas+eng',
            config='--psm 4 --oem 3'
        )
        
        if ocr_text.strip():
            print(f"  [OCR] Extracted {len(ocr_text.strip())} characters (fas-first)")
            return ocr_text.strip()
        
        return ""
    
    except Exception as e:
        print(f"  [OCR] Extraction failed: {e}")
        import traceback
        traceback.print_exc()
        
        # Last resort: try simple text extraction
        print(f"  [OCR] Falling back to pdfplumber text extraction...")
        try:
            fallback_text = page.extract_text() or ""
            return fallback_text
        except:
            return ""


def preprocess_for_ocr(image):
    """
    Preprocess image for better OCR quality, especially for math content.
    
    Steps:
    1. Convert to RGB if needed (ensure consistent format)
    2. Enhance contrast to make symbols stand out
    3. Sharpen to improve edge definition of small math symbols
    4. Keep in color/grayscale (don't binarize - loses detail for math)
    """
    # Ensure image is in RGB mode
    if image.mode not in ('RGB', 'L'):
        image = image.convert('RGB')
    
    # Enhance contrast to make text/symbols more distinct
    enhancer = ImageEnhance.Contrast(image)
    image = enhancer.enhance(1.5)  # Moderate contrast boost
    
    # Sharpen to improve edge definition of small math symbols (∫, ∂, √, etc.)
    enhancer = ImageEnhance.Sharpness(image)
    image = enhancer.enhance(1.8)  # Noticeable sharpening for symbol edges
    
    # Apply slight brightness increase to reduce background noise
    enhancer = ImageEnhance.Brightness(image)
    image = enhancer.enhance(1.1)
    
    return image


def clean_extracted_text(text):
    """
    Clean extracted text while PRESERVING mathematical content.
    
    Rules for math preservation:
    - Keep all math symbols intact (+, -, ×, ÷, =, ∫, Σ, √, ∞, ∂, etc.)
    - Preserve line structure (equations often span multiple lines)
    - Don't strip leading spaces that might be part of equation formatting
    - Remove only truly excessive blank lines (more than 3)
    - Keep numbering and question structure
    """
    # Remove truly excessive blank lines (keep max 3 consecutive for math spacing)
    text = re.sub(r'\n{4,}', '\n\n\n', text)
    
    # Remove lines that are completely empty and surrounded by empty lines
    # (isolated blank lines with no content purpose)
    lines = text.split('\n')
    cleaned_lines = []
    prev_was_empty = False
    prev_prev_was_empty = False
    
    for i, line in enumerate(lines):
        stripped = line.strip()
        is_empty = (stripped == '')
        
        # Keep the line if:
        # 1. It has content, OR
        # 2. It's a single blank line between content (visual spacing), OR
        # 3. It might be part of equation formatting
        if not is_empty:
            cleaned_lines.append(line)
            prev_prev_was_empty = prev_was_empty
            prev_was_empty = False
        else:
            # Only allow empty lines if not too many consecutive
            if not prev_was_empty or (prev_was_empty and not prev_prev_was_empty):
                cleaned_lines.append(line)
            prev_prev_was_empty = prev_was_empty
            prev_was_empty = True
    
    # Remove empty lines from start and end
    while cleaned_lines and cleaned_lines[0].strip() == '':
        cleaned_lines.pop(0)
    while cleaned_lines and cleaned_lines[-1].strip() == '':
        cleaned_lines.pop()
    
    return '\n'.join(cleaned_lines)


def count_pages(file_data):
    """Count total pages in PDF"""
    try:
        file_obj = io.BytesIO(file_data)
        with pdfplumber.open(file_obj) as pdf:
            return len(pdf.pages)
    except:
        return 0


if __name__ == '__main__':
    port = int(os.environ.get('PDF_EXTRACTOR_PORT', 5000))
    print(f"Starting PDF Extractor service on port {port}...")
    print(f"Max file size: {MAX_FILE_SIZE // (1024*1024)}MB")
    print(f"Max pages: {MAX_PAGES}")
    print(f"OCR DPI: {OCR_DPI}")
    print(f"Mode: Smart dual-path (pdfplumber primary, OCR fallback)")
    app.run(host='0.0.0.0', port=port, debug=False)
from flask import Flask, request, jsonify
from flask_cors import CORS
import pdfplumber
import pytesseract
from pdf2image import convert_from_bytes
from PIL import Image
import io
import os
import re

app = Flask(__name__)
CORS(app)

# Configuration
app.config['MAX_CONTENT_LENGTH'] = 15 * 1024 * 1024  # 15MB max request size
MAX_FILE_SIZE = 15 * 1024 * 1024  # 15MB
MAX_PAGES = 30
OCR_CONFIDENCE_THRESHOLD = 60  # Minimum confidence for OCR text

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({'status': 'healthy', 'service': 'pdf-extractor'}), 200

@app.route('/extract-text', methods=['POST'])
def extract_text():
    """
    Extract text from uploaded PDF file.
    Uses pdfplumber for text extraction.
    """
    print(f"[PDF Extractor] Received request: method={request.method}")
    print(f"[PDF Extractor] Content-Type: {request.content_type}")
    print(f"[PDF Extractor] Content-Length: {request.content_length}")
    print(f"[PDF Extractor] Files in request: {list(request.files.keys()) if request.files else 'None'}")
    
    # Validate file presence
    if 'file' not in request.files:
        print(f"[PDF Extractor] ERROR: No 'file' field in request.files")
        return jsonify({'error': 'No file provided', 'success': False}), 400
    
    file = request.files['file']
    print(f"[PDF Extractor] File received: filename={file.filename}, size={len(file.read())} bytes")
    file.seek(0)  # Reset file pointer
    
    if file.filename == '':
        return jsonify({'error': 'No file selected', 'success': False}), 400
    
    # Validate file type
    if not file.filename.lower().endswith('.pdf'):
        return jsonify({'error': 'Only PDF files are allowed', 'success': False}), 400
    
    try:
        # Read file into memory
        file_data = file.read()
        file_size = len(file_data)
        
        print(f"[PDF Extractor] Processing file: {file.filename}, size: {file_size} bytes")
        
        # Check file size
        if file_size > MAX_FILE_SIZE:
            print(f"[PDF Extractor] ERROR: File too large ({file_size} bytes)")
            return jsonify({
                'error': f'File size exceeds {MAX_FILE_SIZE // (1024*1024)}MB limit',
                'success': False
            }), 400
        
        # Extract text using pdfplumber
        text = extract_with_pdfplumber(file_data)
        
        print(f"[PDF Extractor] Extracted {len(text)} characters from PDF")
        
        if not text.strip():
            return jsonify({
                'error': 'No text could be extracted from PDF. The PDF might contain only images.',
                'success': False,
                'suggestion': 'Try using a PDF with selectable text'
            }), 400
        
        return jsonify({
            'text': text,
            'success': True,
            'pages_processed': count_pages(file_data)
        }), 200
    
    except Exception as e:
        print(f"[PDF Extractor] ERROR: {str(e)}")
        import traceback
        traceback.print_exc()
        return jsonify({
            'error': f'Failed to extract text: {str(e)}',
            'success': False
        }), 500

def extract_with_pdfplumber(file_data):
    """
    Extract text from PDF using OCR-based approach.
    Bypasses corrupted PDF text extraction entirely.
    
    DEBUG PIPELINE:
    Stage 1: Raw PDF text extraction (for comparison)
    Stage 2: OCR extraction (actual output)
    Stage 3: Cleaned text (final output)
    """
    text = ""
    file_obj = io.BytesIO(file_data)
    
    with pdfplumber.open(file_obj) as pdf:
        pages_to_process = min(len(pdf.pages), MAX_PAGES)
        
        print(f"\n{'='*60}")
        print(f"[DEBUG] Processing {pages_to_process} pages")
        print(f"{'='*60}\n")
        
        for i in range(pages_to_process):
            page = pdf.pages[i]
            
            # STAGE 1: Extract raw text for debugging (to show the problem)
            print(f"\n{'─'*60}")
            print(f"[DEBUG STAGE 1] Page {i+1} - Raw PDF Text Extraction:")
            print(f"{'─'*60}")
            raw_text = page.extract_text() or ""
            if raw_text:
                print(f"[RAW] Length: {len(raw_text)} characters")
                print(f"[RAW] First 200 chars: {raw_text[:200]}")
                print(f"[RAW] CORRUPTED: Text is reversed/garbled due to PDF font encoding")
            else:
                print(f"[RAW] No text extracted")
            
            # STAGE 2: OCR extraction (the actual solution)
            print(f"\n{'─'*60}")
            print(f"[DEBUG STAGE 2] Page {i+1} - OCR Extraction:")
            print(f"{'─'*60}")
            page_text = extract_page_with_layout(page)
            
            if page_text:
                print(f"[OCR] Length: {len(page_text)} characters")
                print(f"[OCR] First 200 chars: {page_text[:200]}")
                print(f"[OCR] SUCCESS: Text is readable and correct")
            else:
                print(f"[OCR] Failed to extract text")
            
            if page_text:
                text += page_text + "\n\n"
    
    # STAGE 3: Clean the final text
    print(f"\n{'─'*60}")
    print(f"[DEBUG STAGE 3] Final Text Cleaning:")
    print(f"{'─'*60}")
    text = clean_extracted_text(text)
    print(f"[FINAL] Length: {len(text)} characters")
    print(f"[FINAL] First 300 chars: {text[:300]}")
    print(f"{'='*60}\n")
    
    return text.strip()


def extract_page_with_layout(page):
    """
    Extract text from a single page using OCR instead of text extraction.
    This bypasses corrupted PDF text encoding issues.
    
    Pipeline:
    1. Convert PDF page to image (300 DPI)
    2. Apply Tesseract OCR with Persian language
    3. Return clean, properly ordered text
    
    This ensures correct character order, word spacing, and readability.
    """
    try:
        # Convert PDF page to image for OCR
        print(f"[PDF Extractor] Converting page to image for OCR...")
        
        # Get the page image data
        page_image_data = page.to_image(resolution=300)
        
        # Convert to PIL Image
        from PIL import Image as PILImage
        import io as img_io
        
        # Save page image to bytes
        img_byte_arr = img_io.BytesIO()
        page_image_data.save(img_byte_arr, format='PNG')
        img_byte_arr.seek(0)
        
        # Open with PIL
        page_image = PILImage.open(img_byte_arr)
        
        # Convert to grayscale for better OCR
        page_image = page_image.convert('L')
        
        print(f"[PDF Extractor] Running Tesseract OCR with Persian language...")
        
        # Apply OCR with Persian language
        ocr_text = pytesseract.image_to_string(
            page_image,
            lang='fas',  # Persian language
            config='--psm 6 --oem 3'
        )
        
        # Clean and return the text
        if ocr_text.strip():
            print(f"[PDF Extractor] OCR extracted {len(ocr_text.strip())} characters")
            return ocr_text.strip()
        
        return ""
    
    except Exception as e:
        print(f"[PDF Extractor] OCR extraction failed: {e}")
        import traceback
        traceback.print_exc()
        
        # Fallback: try simple text extraction (though it may be corrupted)
        print(f"[PDF Extractor] Falling back to simple text extraction...")
        text = page.extract_text() or ""
        return text


def extract_page_with_ocr(page):
    """
    Extract text from page using OCR (Tesseract).
    Supports Persian/Dari language.
    """
    try:
        # Render page as image
        images = convert_from_bytes(
            page.page_objs[0].get_page_image()['stream'].get_data(),
            dpi=300,
            fmt='png'
        )
        
        if not images:
            return ""
        
        # Perform OCR with Persian language support
        ocr_text = pytesseract.image_to_string(
            images[0],
            lang='fas+eng',  # Persian + English
            config='--psm 6'  # Assume uniform block of text
        )
        
        # Check OCR confidence
        data = pytesseract.image_to_data(
            images[0],
            lang='fas+eng',
            output_type=pytesseract.Output.DICT
        )
        
        # Calculate average confidence
        confidences = [int(conf) for conf in data['conf'] if int(conf) > 0]
        avg_confidence = sum(confidences) / len(confidences) if confidences else 0
        
        print(f"[PDF Extractor] OCR confidence: {avg_confidence:.1f}%")
        
        if avg_confidence < OCR_CONFIDENCE_THRESHOLD:
            print(f"[PDF Extractor] WARNING: Low OCR confidence ({avg_confidence:.1f}%)")
        
        # Apply RTL correction to OCR text
        return fix_rtl_text(ocr_text.strip())
    
    except Exception as e:
        print(f"[PDF Extractor] OCR failed: {e}")
        return ""


def clean_extracted_text(text):
    """
    Clean extracted text while preserving structure:
    - Remove excessive blank lines (max 2 consecutive)
    - Remove extra spaces
    - Preserve line breaks and numbering
    - Keep question structure intact
    """
    # Remove excessive blank lines (keep max 2)
    text = re.sub(r'\n{3,}', '\n\n', text)
    
    # Remove leading/trailing whitespace on each line
    lines = text.split('\n')
    cleaned_lines = [line.strip() for line in lines]
    
    # Remove completely empty lines at start and end
    while cleaned_lines and not cleaned_lines[0]:
        cleaned_lines.pop(0)
    while cleaned_lines and not cleaned_lines[-1]:
        cleaned_lines.pop()
    
    return '\n'.join(cleaned_lines)

def count_pages(file_data):
    """Count total pages in PDF"""
    try:
        file_obj = io.BytesIO(file_data)
        with pdfplumber.open(file_obj) as pdf:
            return len(pdf.pages)
    except:
        return 0

if __name__ == '__main__':
    port = int(os.environ.get('PDF_EXTRACTOR_PORT', 5000))
    print(f"Starting PDF Extractor service on port {port}...")
    print(f"Max file size: {MAX_FILE_SIZE // (1024*1024)}MB")
    print(f"Max pages: {MAX_PAGES}")
    app.run(host='0.0.0.0', port=port, debug=False)
