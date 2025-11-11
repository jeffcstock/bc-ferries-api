CREATE TABLE capacity_routes (
    route_code VARCHAR(6) PRIMARY KEY,
    from_terminal_code VARCHAR(3) NOT NULL,
    to_terminal_code VARCHAR(3) NOT NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    sailing_duration VARCHAR(7) NOT NULL,
    sailings JSONB NOT NULL
);

CREATE TABLE non_capacity_routes (
    route_code VARCHAR(6) PRIMARY KEY,
    from_terminal_code VARCHAR(3) NOT NULL,
    to_terminal_code VARCHAR(3) NOT NULL,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    sailing_duration VARCHAR(7) NOT NULL,
    sailings JSONB NOT NULL
);