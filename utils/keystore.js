/*******************************************************************************
 * Copyright (c) 2016 IBM Corp.
 *
 * All rights reserved. 
 *
 * Contributors:
 *   Justin E. Ervin - Initial implementation
 *******************************************************************************/

var keystore_creds = {
    host: '',
    port: '',
    username: '',
    password: '',
    database: '',
};

var db;

function setupConn(credentials, cb) {
    keystore_creds = {
        host: credentials.host,
        port: credentials.port,
        username: credentials.username,
        password: credentials.password,
        database: credentials.database,
    };

    console.log("[KeyValStore] Connecting to the database server...");
    var nano = require('nano')('https://' + keystore_creds.username + ':' + keystore_creds.password + '@' + keystore_creds.host + ':' + keystore_creds.port);	//lets use api key to make the docs

    nano.db.get(keystore_creds.database, function (geterr, getbody) {
        if (!geterr) {
            console.log("[KeyValStore] Ready. Found database");
            db = nano.use(keystore_creds.database);

            if (cb) {
                cb(null, true);
            }
        } else if (geterr.error == 'not_found') {
            console.log("[KeyValStore] Creating database...");
            nano.db.create(keystore_creds.database, function (createerr, createbody) {
                if (!createerr) {
                    console.log("[KeyValStore] Ready. Created database");
                    db = nano.use(keystore_creds.database);

                    if (cb) {
                        cb(null, true);
                    }
                } else if (cb) {
                    cb(null, false);
                }
            });
        } else if (cb) {
            cb(null, false);
        }
    });
}

/**
 * A local file-based key value store.
 * This implements the KeyValStore interface.
 */
var CouchDBKeyValStore = (function () {
    function CouchDBKeyValStore() {

    }

    CouchDBKeyValStore.prototype.setValue = function (name, value, cb) {
        get_user_doc(name, function (currerr, currmemberdoc) {
            var memberdoc = JSON.parse(value);
            memberdoc._id = name;

            if (currerr != null) {
                insert_doc(memberdoc, function (err, userdoc) {
                    if (err) {
                        console.log(err);
                    }
                    cb(err);
                });
            } else {
                memberdoc._rev = currmemberdoc._rev;
                insert_doc(memberdoc, function (err, userdoc) {
                    if (err) {
                        console.log(err);
                    }
                    cb(err);
                });
            }
        });
    };

    /**
     * Get the value associated with name.
     * @param name
     * @param cb function(err,value)
     */
    CouchDBKeyValStore.prototype.getValue = function (name, cb) {
        get_user_doc(name, function (err, memberdoc) {
            if (err != null) {
                if (memberdoc.error != 'not_found') {
                    return cb(memberdoc);
                }
                return cb(null, null);
            } else {
                return cb(null, JSON.stringify(memberdoc));
            }
        });
    };

    return CouchDBKeyValStore;
} ()); // end CouchDBKeyValStore

function get_user_doc(username, cb) {
    if (db) {
        db.get(username, { revs_info: true }, function (err, body) {		//doc_name, query parameters, callback
            if (cb) {
                if (!err && body) cb(null, body);
                else if (err && err.statusCode) cb(err.statusCode, { error: err.error, reason: err.reason });
                else cb(500, { error: err, reason: 'unknown!' });
            }
        });
    } else {
        cb(500, { error: 'connect', reason: 'failed to connect database!' });
    }
}

function insert_doc(doc, cb) {
    if (db) {
        db.insert(doc, function (err, body) {
            if (cb) {
                if (!err && body) {
                    doc._rev = body.rev;
                    cb(null, doc);
                }
                else if (err && err.statusCode) cb(err.statusCode, { error: err.error, reason: err.reason });
                else cb(500, { error: err, reason: 'unknown!' });
            }
        });
    } else {
        cb(500, { error: 'connect', reason: 'failed to connect database!' });
    }
}

function newCouchDBKeyValStore() {
    return new CouchDBKeyValStore();
}

module.exports.newCouchDBKeyValStore = newCouchDBKeyValStore;
module.exports.setupConn = setupConn;