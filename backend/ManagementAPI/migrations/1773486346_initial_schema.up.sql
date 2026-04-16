CREATE TABLE users
(
    id            uuid primary key not null default gen_random_uuid(),
    username      text unique      not null,
    password_hash text             null -- if null, no login via username/password possible
);
