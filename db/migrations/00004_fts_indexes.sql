-- +goose Up
-- FTS5 full-text search indexes (external content mode, compatible with UUID PKs).

CREATE VIRTUAL TABLE IF NOT EXISTS fts_companies USING fts5(
    name, domain, industry, notes
);
CREATE VIRTUAL TABLE IF NOT EXISTS fts_contacts USING fts5(
    first_name, last_name, email, phone, job_title, notes
);
CREATE VIRTUAL TABLE IF NOT EXISTS fts_leads USING fts5(
    display_name, first_name, last_name, email, phone, company_name, notes
);
CREATE VIRTUAL TABLE IF NOT EXISTS fts_deals USING fts5(
    name, description
);
CREATE VIRTUAL TABLE IF NOT EXISTS fts_tasks USING fts5(
    title, description
);
CREATE VIRTUAL TABLE IF NOT EXISTS fts_projects USING fts5(
    name, description
);

-- +goose Down
DROP TABLE IF EXISTS fts_projects;
DROP TABLE IF EXISTS fts_tasks;
DROP TABLE IF EXISTS fts_deals;
DROP TABLE IF EXISTS fts_leads;
DROP TABLE IF EXISTS fts_contacts;
DROP TABLE IF EXISTS fts_companies;
