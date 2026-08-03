# PDF Extractor Setup Guide

## Overview
The PDF Extractor now uses advanced text extraction with:
- ✅ **pdfplumber** for layout-preserving text extraction
- ✅ **Tesseract OCR** for scanned PDFs (Persian/Dari support)
- ✅ Line-by-line extraction with structure preservation
- ✅ Automatic fallback to OCR when text extraction fails

---

## Installation

### Step 1: Install Python Dependencies

```bash
cd e:\Kankor\backend\pdf-extractor
pip install -r requirements.txt
```

### Step 2: Install Tesseract OCR

**Windows:**
1. Download Tesseract installer: https://github.com/UB-Mannheim/tesseract/wiki
2. Install to: `C:\Program Files\Tesseract-OCR`
3. Add to system PATH:
   - Open System Properties → Environment Variables
   - Edit PATH, add: `C:\Program Files\Tesseract-OCR`
   - Click OK

**Verify installation:**
```bash
tesseract --version
```

### Step 3: Install Poppler (for PDF to Image conversion)

**Windows:**
1. Download: https://github.com/oschwartz10612/poppler-windows/releases
2. Extract to: `C:\poppler`
3. Add to system PATH:
   - Add: `C:\poppler\Library\bin` to PATH

**Verify installation:**
```bash
pdfinfo -v
```

### Step 4: Download Persian Language Data for Tesseract

1. Download Persian trained data: https://github.com/tesseract-ocr/tessdata/raw/main/fas.traineddata
2. Place it in Tesseract's tessdata folder:
   - `C:\Program Files\Tesseract-OCR\tessdata\fas.traineddata`

**Verify:**
```bash
tesseract --list-langs
```
You should see `fas` in the list.

---

## Testing

### Test 1: Health Check
```bash
curl http://localhost:8000/health
```

Expected:
```json
{"status": "healthy", "service": "pdf-extractor"}
```

### Test 2: Extract Text from PDF
```bash
curl -X POST http://localhost:8000/extract-text \
  -F "file=@your-test.pdf"
```

---

## How It Works

### Extraction Flow:
```
1. PDF Upload
   ↓
2. Try pdfplumber layout extraction
   ├─ Success → Clean text
   └─ Failed/Empty → Try OCR
      ↓
3. OCR with Tesseract (Persian + English)
   ├─ Confidence > 60% → Use OCR text
   └─ Confidence < 60% → Warning logged
   ↓
4. Clean extracted text
   ├─ Remove extra blank lines
   ├─ Preserve line breaks
   ├─ Keep numbering structure
   └─ Return structured text
```

### Features:

#### 1. **Layout-Preserving Extraction**
- Extracts words with coordinates
- Groups words by line (y-coordinate)
- Sorts lines top-to-bottom, left-to-right
- Preserves question numbering and structure

#### 2. **OCR Fallback**
- Automatically detects scanned PDFs
- Uses Tesseract with Persian language (`fas`)
- Calculates confidence score
- Warns if confidence is low (<60%)

#### 3. **Text Cleaning**
- Removes excessive blank lines (max 2)
- Removes extra spaces
- Preserves line breaks
- Keeps numbering intact
- Maintains question structure

---

## Configuration

### Environment Variables (Optional)
```bash
PDF_EXTRACTOR_PORT=8000  # Default: 8000
```

### Constants in main.py
```python
MAX_FILE_SIZE = 15 * 1024 * 1024  # 15MB
MAX_PAGES = 30                     # Max pages to process
OCR_CONFIDENCE_THRESHOLD = 60      # Min OCR confidence %
```

---

## Troubleshooting

### Issue: "Tesseract not found"
**Solution:**
- Verify Tesseract is installed
- Check PATH includes Tesseract directory
- Restart terminal/IDE after adding to PATH

### Issue: "poppler not found"
**Solution:**
- Verify Poppler is extracted
- Check PATH includes `poppler\Library\bin`
- Restart terminal after adding to PATH

### Issue: "fas language not found"
**Solution:**
- Download `fas.traineddata`
- Place in `C:\Program Files\Tesseract-OCR\tessdata\`
- Verify with: `tesseract --list-langs`

### Issue: Low OCR confidence
**Solutions:**
- Use higher quality PDF (300+ DPI)
- Ensure PDF is not heavily compressed
- Try PDF with selectable text instead of scanned images

---

## Performance Tips

1. **Text-based PDFs**: Much faster than OCR (instant)
2. **Scanned PDFs**: OCR takes 2-5 seconds per page
3. **Large PDFs**: Limit to 30 pages max
4. **DPI**: 300 DPI is optimal for OCR

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| flask | 3.0.0 | Web framework |
| flask-cors | 4.0.0 | CORS support |
| pdfplumber | 0.10.3 | PDF text extraction |
| pytesseract | 0.3.10 | Tesseract OCR wrapper |
| Pillow | 10.2.0 | Image processing |
| pdf2image | 1.17.0 | PDF to image conversion |

---

## Logs to Monitor

When running, watch for:
```
[PDF Extractor] Page X: No text found, attempting OCR...
[PDF Extractor] OCR confidence: 85.3%
[PDF Extractor] WARNING: Low OCR confidence (45.2%)
[PDF Extractor] Extracted 47105 characters from PDF
```

---

## Next Steps

After setup, the PDF extractor will:
1. ✅ Extract text with layout preservation
2. ✅ Handle scanned PDFs with OCR
3. ✅ Support Persian/Dari language
4. ✅ Return clean, structured text
5. ✅ Send to Gemini AI for question extraction
