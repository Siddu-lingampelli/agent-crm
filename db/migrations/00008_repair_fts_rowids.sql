-- +goose Up
-- Preserve entity rowids when rebuilding and maintaining search indexes.
DROP TRIGGER IF EXISTS trg_companies_ai;
DROP TRIGGER IF EXISTS trg_companies_au;
DROP TRIGGER IF EXISTS trg_companies_ad;
DELETE FROM fts_companies;
INSERT INTO fts_companies(rowid,name,domain,industry,notes) SELECT rowid,name,domain,industry,notes FROM companies;
-- +goose StatementBegin
CREATE TRIGGER trg_companies_ai AFTER INSERT ON companies BEGIN
INSERT INTO fts_companies(rowid,name,domain,industry,notes) VALUES(new.rowid,new.name,new.domain,new.industry,new.notes);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_companies_au AFTER UPDATE ON companies BEGIN
DELETE FROM fts_companies WHERE rowid=old.rowid;
INSERT INTO fts_companies(rowid,name,domain,industry,notes) VALUES(new.rowid,new.name,new.domain,new.industry,new.notes);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_companies_ad AFTER DELETE ON companies BEGIN
DELETE FROM fts_companies WHERE rowid=old.rowid;
END;
-- +goose StatementEnd
DROP TRIGGER IF EXISTS trg_contacts_ai;
DROP TRIGGER IF EXISTS trg_contacts_au;
DROP TRIGGER IF EXISTS trg_contacts_ad;
DELETE FROM fts_contacts;
INSERT INTO fts_contacts(rowid,first_name,last_name,email,phone,job_title,notes) SELECT rowid,first_name,last_name,email,phone,job_title,notes FROM contacts;
-- +goose StatementBegin
CREATE TRIGGER trg_contacts_ai AFTER INSERT ON contacts BEGIN
INSERT INTO fts_contacts(rowid,first_name,last_name,email,phone,job_title,notes) VALUES(new.rowid,new.first_name,new.last_name,new.email,new.phone,new.job_title,new.notes);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_contacts_au AFTER UPDATE ON contacts BEGIN
DELETE FROM fts_contacts WHERE rowid=old.rowid;
INSERT INTO fts_contacts(rowid,first_name,last_name,email,phone,job_title,notes) VALUES(new.rowid,new.first_name,new.last_name,new.email,new.phone,new.job_title,new.notes);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_contacts_ad AFTER DELETE ON contacts BEGIN
DELETE FROM fts_contacts WHERE rowid=old.rowid;
END;
-- +goose StatementEnd
DROP TRIGGER IF EXISTS trg_leads_ai;
DROP TRIGGER IF EXISTS trg_leads_au;
DROP TRIGGER IF EXISTS trg_leads_ad;
DELETE FROM fts_leads;
INSERT INTO fts_leads(rowid,display_name,first_name,last_name,email,phone,company_name,notes) SELECT rowid,display_name,first_name,last_name,email,phone,company_name,description FROM leads;
-- +goose StatementBegin
CREATE TRIGGER trg_leads_ai AFTER INSERT ON leads BEGIN
INSERT INTO fts_leads(rowid,display_name,first_name,last_name,email,phone,company_name,notes) VALUES(new.rowid,new.display_name,new.first_name,new.last_name,new.email,new.phone,new.company_name,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_leads_au AFTER UPDATE ON leads BEGIN
DELETE FROM fts_leads WHERE rowid=old.rowid;
INSERT INTO fts_leads(rowid,display_name,first_name,last_name,email,phone,company_name,notes) VALUES(new.rowid,new.display_name,new.first_name,new.last_name,new.email,new.phone,new.company_name,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_leads_ad AFTER DELETE ON leads BEGIN
DELETE FROM fts_leads WHERE rowid=old.rowid;
END;
-- +goose StatementEnd
DROP TRIGGER IF EXISTS trg_deals_ai;
DROP TRIGGER IF EXISTS trg_deals_au;
DROP TRIGGER IF EXISTS trg_deals_ad;
DELETE FROM fts_deals;
INSERT INTO fts_deals(rowid,name,description) SELECT rowid,name,description FROM deals;
-- +goose StatementBegin
CREATE TRIGGER trg_deals_ai AFTER INSERT ON deals BEGIN
INSERT INTO fts_deals(rowid,name,description) VALUES(new.rowid,new.name,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_deals_au AFTER UPDATE ON deals BEGIN
DELETE FROM fts_deals WHERE rowid=old.rowid;
INSERT INTO fts_deals(rowid,name,description) VALUES(new.rowid,new.name,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_deals_ad AFTER DELETE ON deals BEGIN
DELETE FROM fts_deals WHERE rowid=old.rowid;
END;
-- +goose StatementEnd
DROP TRIGGER IF EXISTS trg_tasks_ai;
DROP TRIGGER IF EXISTS trg_tasks_au;
DROP TRIGGER IF EXISTS trg_tasks_ad;
DELETE FROM fts_tasks;
INSERT INTO fts_tasks(rowid,title,description) SELECT rowid,title,description FROM tasks;
-- +goose StatementBegin
CREATE TRIGGER trg_tasks_ai AFTER INSERT ON tasks BEGIN
INSERT INTO fts_tasks(rowid,title,description) VALUES(new.rowid,new.title,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_tasks_au AFTER UPDATE ON tasks BEGIN
DELETE FROM fts_tasks WHERE rowid=old.rowid;
INSERT INTO fts_tasks(rowid,title,description) VALUES(new.rowid,new.title,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_tasks_ad AFTER DELETE ON tasks BEGIN
DELETE FROM fts_tasks WHERE rowid=old.rowid;
END;
-- +goose StatementEnd
DROP TRIGGER IF EXISTS trg_projects_ai;
DROP TRIGGER IF EXISTS trg_projects_au;
DROP TRIGGER IF EXISTS trg_projects_ad;
DELETE FROM fts_projects;
INSERT INTO fts_projects(rowid,name,description) SELECT rowid,name,description FROM projects;
-- +goose StatementBegin
CREATE TRIGGER trg_projects_ai AFTER INSERT ON projects BEGIN
INSERT INTO fts_projects(rowid,name,description) VALUES(new.rowid,new.name,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_projects_au AFTER UPDATE ON projects BEGIN
DELETE FROM fts_projects WHERE rowid=old.rowid;
INSERT INTO fts_projects(rowid,name,description) VALUES(new.rowid,new.name,new.description);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_projects_ad AFTER DELETE ON projects BEGIN
DELETE FROM fts_projects WHERE rowid=old.rowid;
END;
-- +goose StatementEnd

-- +goose Down
-- Correct rowid mappings are retained on rollback to avoid corrupting search.
SELECT 1;
