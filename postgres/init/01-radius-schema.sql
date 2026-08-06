CREATE TABLE IF NOT EXISTS radcheck (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op CHAR(2) NOT NULL DEFAULT '==',
    value VARCHAR(253) NOT NULL DEFAULT ''
);

CREATE INDEX idx_radcheck_username ON radcheck(username);

CREATE TABLE IF NOT EXISTS radreply (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op CHAR(2) NOT NULL DEFAULT '=',
    value VARCHAR(253) NOT NULL DEFAULT ''
);

CREATE INDEX idx_radreply_username ON radreply(username);

CREATE TABLE IF NOT EXISTS radgroupcheck (
    id SERIAL PRIMARY KEY,
    groupname VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op CHAR(2) NOT NULL DEFAULT '==',
    value VARCHAR(253) NOT NULL DEFAULT ''
);

CREATE INDEX idx_radgroupcheck_groupname ON radgroupcheck(groupname);

CREATE TABLE IF NOT EXISTS radgroupreply (
    id SERIAL PRIMARY KEY,
    groupname VARCHAR(64) NOT NULL DEFAULT '',
    attribute VARCHAR(64) NOT NULL DEFAULT '',
    op CHAR(2) NOT NULL DEFAULT '=',
    value VARCHAR(253) NOT NULL DEFAULT ''
);

CREATE INDEX idx_radgroupreply_groupname ON radgroupreply(groupname);

CREATE TABLE IF NOT EXISTS radusergroup (
    id SERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL DEFAULT '',
    groupname VARCHAR(64) NOT NULL DEFAULT '',
    priority INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_radusergroup_username ON radusergroup(username);

CREATE TABLE IF NOT EXISTS radacct (
    radacctid BIGSERIAL PRIMARY KEY,
    acctsessionid VARCHAR(64) NOT NULL DEFAULT '',
    acctuniqueid VARCHAR(32) NOT NULL DEFAULT '',
    username VARCHAR(64) NOT NULL DEFAULT '',
    groupname VARCHAR(64) NOT NULL DEFAULT '',
    realm VARCHAR(64) NOT NULL DEFAULT '',
    nasipaddress INET NOT NULL,
    nasportid VARCHAR(15) NOT NULL DEFAULT '',
    nasporttype VARCHAR(32) NOT NULL DEFAULT '',
    acctstarttime TIMESTAMP WITH TIME ZONE,
    acctupdatetime TIMESTAMP WITH TIME ZONE,
    acctstoptime TIMESTAMP WITH TIME ZONE,
    acctinterval INTEGER DEFAULT NULL,
    acctsessiontime INTEGER DEFAULT NULL,
    acctauthentic VARCHAR(32) NOT NULL DEFAULT '',
    connectinfo_start VARCHAR(50) NOT NULL DEFAULT '',
    connectinfo_stop VARCHAR(50) NOT NULL DEFAULT '',
    acctinputoctets BIGINT DEFAULT NULL,
    acctoutputoctets BIGINT DEFAULT NULL,
    calledstationid VARCHAR(50) NOT NULL DEFAULT '',
    callingstationid VARCHAR(50) NOT NULL DEFAULT '',
    acctterminatecause VARCHAR(32) NOT NULL DEFAULT '',
    servicetype VARCHAR(32) NOT NULL DEFAULT '',
    framedprotocol VARCHAR(32) NOT NULL DEFAULT '',
    framedipaddress INET,
    framedipv6address INET,
    framedipv6prefix INET,
    framedinterfaceid VARCHAR(44) NOT NULL DEFAULT '',
    delegatedipv6prefix INET
);

CREATE INDEX idx_radacct_username ON radacct(username);
CREATE INDEX idx_radacct_acctsessionid ON radacct(acctsessionid);
CREATE INDEX idx_radacct_acctuniqueid ON radacct(acctuniqueid);
CREATE INDEX idx_radacct_acctstarttime ON radacct(acctstarttime);
CREATE INDEX idx_radacct_acctstoptime ON radacct(acctstoptime);
CREATE INDEX idx_radacct_nasipaddress ON radacct(nasipaddress);
CREATE INDEX idx_radacct_framedipaddress ON radacct(framedipaddress);

CREATE TABLE IF NOT EXISTS radpostauth (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL DEFAULT '',
    pass VARCHAR(64) NOT NULL DEFAULT '',
    reply VARCHAR(32) NOT NULL DEFAULT '',
    authdate TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX idx_radpostauth_username ON radpostauth(username);
CREATE INDEX idx_radpostauth_authdate ON radpostauth(authdate);

CREATE TABLE IF NOT EXISTS nas (
    id SERIAL PRIMARY KEY,
    nasname VARCHAR(128) NOT NULL,
    shortname VARCHAR(32) NOT NULL DEFAULT '',
    type VARCHAR(30) NOT NULL DEFAULT 'other',
    ports INTEGER NOT NULL DEFAULT 0,
    secret VARCHAR(60) NOT NULL DEFAULT 'secret',
    server VARCHAR(64) NOT NULL DEFAULT '',
    community VARCHAR(50) NOT NULL DEFAULT '',
    description VARCHAR(200) NOT NULL DEFAULT ''
);

CREATE INDEX idx_nas_nasname ON nas(nasname);
