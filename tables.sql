CREATE TABLE IF NOT EXISTS trigger(
    id VARCHAR,
    timestamp DATETIME,
    type VARCHAR,
    intensity INT CHECK(intensity >= 0 AND intensity <= 10),
    compulsion BOOL,
    notes VARCHAR
);

CREATE TABLE IF NOT EXISTS action(
    id VARCHAR,
    timestamp DATETIME,
    type VARCHAR,
    relief BOOL,
    duration INTEGER,
    notes VARCHAR
);

CREATE TABLE IF NOT EXISTS overall(
    id VARCHAR,
    timestamp DATETIME,
    clean_streak INT,
    gym_session BOOL,
    coding_hours INT,
    reading_hours INT,
    mood_score INTEGER CHECK(mood_score >= 0 AND mood_score <= 100),
    notes VARCHAR
);