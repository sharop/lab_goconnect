CREATE TABLE IF NOT EXISTS example.user
(
    id bigint default unique_rowid() not null
        constraint "primary"
            primary key,
    name varchar(255),
    salt varchar(255),
    age bigint,
    passwd varchar(200),
    created timestamp,
    updated timestamp
);
