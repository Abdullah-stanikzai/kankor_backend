import psycopg2
import os

# Database connection parameters
DB_HOST = "localhost"
DB_PORT = "5432"
DB_NAME = "kankor_exam"
DB_USER = "postgres"
DB_PASSWORD = "password"

try:
    # Connect to database
    conn = psycopg2.connect(
        host=DB_HOST,
        port=DB_PORT,
        database=DB_NAME,
        user=DB_USER,
        password=DB_PASSWORD
    )
    
    cursor = conn.cursor()
    
    print("=== DATABASE RELATIONSHIP CHECK ===\n")
    
    # Check all centers
    print("1. All Centers:")
    cursor.execute("SELECT id, name, admin_user_id, is_active FROM educational_centers ORDER BY created_at")
    centers = cursor.fetchall()
    
    for center in centers:
        id, name, admin_user_id, is_active = center
        admin_id_str = admin_user_id if admin_user_id else "NULL"
        print(f"  Center: {name} (ID: {id}) | Admin User ID: {admin_id_str} | Active: {is_active}")
    
    print("\n2. All Center Admins:")
    cursor.execute("SELECT id, email, full_name, center_id FROM users WHERE role = 'center_admin' ORDER BY created_at")
    admins = cursor.fetchall()
    
    for admin in admins:
        id, email, full_name, center_id = admin
        center_id_str = center_id if center_id else "NULL"
        print(f"  Admin: {full_name} ({email}) | ID: {id} | Center ID: {center_id_str}")
    
    print("\n3. Centers with Admin Names (JOIN):")
    cursor.execute("""
        SELECT ec.id, ec.name, ec.admin_user_id, u.full_name, u.email
        FROM educational_centers ec 
        LEFT JOIN users u ON ec.admin_user_id = u.id 
        WHERE ec.is_active = true 
        ORDER BY ec.created_at
    """)
    joined_data = cursor.fetchall()
    
    for row in joined_data:
        id, name, admin_user_id, full_name, email = row
        admin_id_str = admin_user_id if admin_user_id else "NULL"
        admin_name_str = full_name if full_name else "NULL"
        print(f"  Center: {name} | Admin ID: {admin_id_str} | Admin Name: {admin_name_str}")
    
    print("\n4. Checking for orphaned relationships:")
    
    # Check for invalid admin_user_id references
    cursor.execute("""
        SELECT ec.id, ec.name, ec.admin_user_id 
        FROM educational_centers ec 
        WHERE ec.admin_user_id IS NOT NULL 
        AND ec.admin_user_id NOT IN (SELECT id FROM users)
    """)
    invalid_admin_refs = cursor.fetchall()
    
    if invalid_admin_refs:
        print("  Centers with invalid admin_user_id:")
        for row in invalid_admin_refs:
            id, name, admin_user_id = row
            print(f"    Center '{name}' references non-existent admin ID: {admin_user_id}")
    else:
        print("  No invalid admin_user_id references found")
    
    # Check for invalid center_id references
    cursor.execute("""
        SELECT u.id, u.full_name, u.center_id 
        FROM users u 
        WHERE u.role = 'center_admin' 
        AND u.center_id IS NOT NULL 
        AND u.center_id NOT IN (SELECT id FROM educational_centers)
    """)
    invalid_center_refs = cursor.fetchall()
    
    if invalid_center_refs:
        print("  Admins with invalid center_id:")
        for row in invalid_center_refs:
            id, full_name, center_id = row
            print(f"    Admin '{full_name}' references non-existent center ID: {center_id}")
    else:
        print("  No invalid center_id references found")
    
    cursor.close()
    conn.close()
    
    print("\n=== CHECK COMPLETE ===")
    
except Exception as e:
    print(f"Error connecting to database: {e}")