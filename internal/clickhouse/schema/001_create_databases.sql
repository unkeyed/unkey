-- +goose up

CREATE DATABASE verifications;
CREATE DATABASE telemetry;
CREATE DATABASE metrics;
CREATE DATABASE ratelimits;
CREATE DATABASE business;
CREATE DATABASE billing;


-- +goose down
DROP DATABASE verifications;
DROP DATABASE telemetry;
DROP DATABASE metrics;
DROP DATABASE ratelimits;
DROP DATABASE business;
DROP DATABASE billing;
