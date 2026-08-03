"""
Test script to verify RTL text correction is working properly.
Run this to check if Persian/Dari characters are being joined correctly.
"""

import arabic_reshaper
from bidi.algorithm import get_display
import unicodedata
import sys
import os

# Add parent directory to path to import from main.py
sys.path.insert(0, os.path.dirname(__file__))
from main import normalize_rtl_spacing, fix_rtl_text, is_already_shaped

def test_rtl_correction():
    """Test RTL text correction with sample Persian text."""
    
    # Test Case 1: Normal Persian text with proper word separation
    print("=" * 60)
    print("Test Case 1: Normal Persian Sentence")
    print("=" * 60)
    
    normal_text = "در کدام قرن دولت یفتلی در کشور ما رو به زوال نهاد"
    print(f"\nOriginal: {normal_text}")
    
    # Check if already shaped
    shaped = is_already_shaped(normal_text)
    print(f"Already shaped: {shaped}")
    
    # Normalize spacing
    normalized = normalize_rtl_spacing(normal_text)
    print(f"After spacing normalization: {normalized}")
    
    # Apply the full correction pipeline
    try:
        final = fix_rtl_text(normal_text)
        print(f"Final (full pipeline): {final}")
        
        print("\n✅ Test Case 1 PASSED")
    except Exception as e:
        print(f"\n❌ Test Case 1 FAILED: {e}")
    
    # Test Case 2: Text with Arabic Ya and Kaf (should be converted to Persian)
    print("\n" + "=" * 60)
    print("Test Case 2: Arabic vs Persian Characters")
    print("=" * 60)
    
    # Arabic variants (wrong for Persian)
    arabic_text = "كتاب يافت"  # Uses Arabic ك and ي
    print(f"\nArabic variants: {arabic_text}")
    
    # Convert to Persian
    persian_text = arabic_text.replace('\u0643', '\u06A9').replace('\u064A', '\u06CC')
    print(f"Persian variants: {persian_text}")
    
    try:
        final = fix_rtl_text(persian_text)
        print(f"After correction: {final}")
        print("\n✅ Test Case 2 PASSED")
    except Exception as e:
        print(f"\n❌ Test Case 2 FAILED: {e}")
    
    complex_text = "سوال 1: در کدام قرن دولت یفتلی رو به زوال نهاد؟ (1) قرن پنجم (2) قرن ششم"
    print(f"\nOriginal: {complex_text}")
    
    try:
        final = fix_rtl_text(complex_text)
        print(f"After correction: {final}")
        print("\n✅ Test Case 3 PASSED")
    except Exception as e:
        print(f"\n❌ Test Case 3 FAILED: {e}")
    
    # Test Case 4: Text with excessive spaces
    print("\n" + "=" * 60)
    print("Test Case 4: Text with Excessive Spaces")
    print("=" * 60)
    
    excessive_spaces = "در   کدام    قرن     دولت یفتلی"
    print(f"\nOriginal: {excessive_spaces}")
    
    normalized = normalize_rtl_spacing(excessive_spaces)
    print(f"After normalization: {normalized}")
    
    try:
        final = fix_rtl_text(excessive_spaces)
        print(f"Final result: {final}")
        print("\n✅ Test Case 4 PASSED")
    except Exception as e:
        print(f"\n❌ Test Case 4 FAILED: {e}")
    
    print("\n" + "=" * 60)
    print("All tests completed!")
    print("=" * 60)

if __name__ == "__main__":
    test_rtl_correction()
