-- Pokemon table
CREATE TABLE IF NOT EXISTS pokemon (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    height REAL,
    weight REAL,
    base_experience INTEGER
);

-- Pokemon stats table
CREATE TABLE IF NOT EXISTS pokemon_stats (
    pokemon_id INTEGER PRIMARY KEY,
    hp INTEGER NOT NULL,
    attack INTEGER NOT NULL,
    defense INTEGER NOT NULL,
    sp_attack INTEGER NOT NULL,
    sp_defense INTEGER NOT NULL,
    speed INTEGER NOT NULL,
    FOREIGN KEY(pokemon_id) REFERENCES pokemon(id)
);

-- Pokemon abilities table
CREATE TABLE IF NOT EXISTS pokemon_abilities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pokemon_id INTEGER NOT NULL,
    ability TEXT NOT NULL,
    is_hidden BOOLEAN DEFAULT 0,
    FOREIGN KEY(pokemon_id) REFERENCES pokemon(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_pokemon_name ON pokemon(name);
CREATE INDEX IF NOT EXISTS idx_pokemon_type ON pokemon(type);
