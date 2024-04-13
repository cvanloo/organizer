create database if not exists organizer
	character set = 'utf8mb4'
	collate = 'utf8mb4_unicode_ci';

use organizer;

create table if not exists migrations (
	version int primary key auto_increment,
	created_at datetime not null default current_timestamp
);

alter table migrations auto_increment = 0;

-- sudo pacman -Syu mariadb
-- sudo systemctl start mariadb
-- sudo mariadb-install-db --user=mysql --basedir=/usr --datadir=/var/lib/mysql
-- sudo mariadb-secure-installation # (answer everything with yes, use unix_socket auth)
-- sudo mariadb
-- create user 'organizer'@'localhost' identified via unix_socket;
-- source create_database.sql
-- grant all privileges on organizer.* to 'organizer'@'localhost';
-- flush privileges;
