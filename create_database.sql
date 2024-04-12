create database if not exists organizer
	character set = 'utf8mb4'
	collate = 'utf8mb4_unicode_ci';

use organizer;

create table if not exists migrations (
	version int primary key auto_increment,
	created_at datetime not null default current_timestamp
);

alter table migrations auto_increment = 0;
