# greenlight

## Prepare Database

    CREATE DATABASE greenlight;
    CREATE ROLE greenlight WITH LOGIN PASSWORD 'pa55word'
    CREATE EXTENSION IF NOT EXISTS citext;
    GRANT ALL PRIVILEGES ON DATABASE greenlight to greenlight;
    GRANT ALL ON SCHEMA public TO greenlight;
    ALTER DATABASE greenlight OWNER TO greenlight;
