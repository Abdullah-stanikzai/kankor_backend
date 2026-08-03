# PDF Extractor Microservice

Extracts text from PDF files for the Kankor Question Bank system.

## Setup

```bash
# Create virtual environment (recommended)
python -m venv venv

# Activate virtual environment
# Windows:
venv\Scripts\activate
# Linux/Mac:
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Run the service
python main.py
```

## API Endpoints

### Health Check
```
GET /health
```

### Extract Text
```
POST /extract-text
Content-Type: multipart/form-data

Form data:
- file: PDF file (max 15MB, max 30 pages)
```

**Response:**
```json
{
  "text": "Extracted text content...",
  "success": true,
  "pages_processed": 5
}
```

## Configuration

- `PDF_EXTRACTOR_PORT`: Port number (default: 5000)
- `MAX_FILE_SIZE`: 15MB
- `MAX_PAGES`: 30 pages
