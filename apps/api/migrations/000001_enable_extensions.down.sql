-- Revert the foundation extension added for case-insensitive email support.

DROP EXTENSION IF EXISTS citext;
