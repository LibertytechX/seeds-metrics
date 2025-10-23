import psycopg2
from psycopg2 import OperationalError

def create_connection():
    """Establish connection to PostgreSQL database."""
    try:
        connection = psycopg2.connect(
            dbname="seedsmetrics",
            user="postgres",
            password="19sedimat54",
            host="localhost",
            port="5432"
        )
        print("✅ Connection to PostgreSQL successful")
        return connection
    except OperationalError as e:
        print("❌ The error occurred:", e)
        return None

def close_connection(connection):
    """Close PostgreSQL connection."""
    if connection:
        connection.close()
        print("🔒 Connection closed")

if __name__ == "__main__":
    conn = create_connection()
    if conn:
        # Example query to verify connection
        cursor = conn.cursor()
        cursor.execute("SELECT current_database(), current_user;")
        db_info = cursor.fetchone()
        print("📘 Connected to database:", db_info[0])
        print("👤 Logged in as:", db_info[1])

        cursor.close()
        close_connection(conn)
