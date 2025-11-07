#!/bin/sh

psql "postgres://admin:asdf@localhost:5432/app?sslmode=disable" -f migrations/001_initial_schema.sql
