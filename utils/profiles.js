/*******************************************************************************
 * Copyright (c) 2016 IBM Corp.
 *
 * All rights reserved. 
 *
 * Contributors:
 *   Justin E. Ervin - Initial implementation
 *******************************************************************************/

var crypto = require('crypto');
var encoding = require('./encoding');
var fs = require('fs');
var profilesStorage = [];
var currChainId;
var currChain;
var profileDir;
var cacheUserProfile = false;
var createUserRSAKeys = false;
var defaultCustomUserProperties = {};
var profiles_creds = {
    host: '',
    port: '',
    username: '',
    password: '',
    database: '',
};
var nano;
var db;

function setupConn(cb) {
    console.log("[Profiles] Connecting to the database server...");
    nano = require('nano')('https://' + profiles_creds.username + ':' + profiles_creds.password + '@' + profiles_creds.host + ':' + profiles_creds.port);

    if (nano) {
        //lets use api key to make the docs
        return nano.db.get(profiles_creds.database, function (geterr, getbody) {
            if (!geterr) {
                console.log("[Profiles] Ready. Found database");
                db = nano.use(profiles_creds.database);
                if (cb) {
                    cb(null, true);
                }
            } else if (geterr.error == 'not_found') {
                console.log("[Profiles] Creating database...");
                return nano.db.create(profiles_creds.database, function (createerr, createbody) {
                    if (!createerr) {
                        console.log("[Profiles] Ready. Created database");
                        db = nano.use(profiles_creds.database);

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
    } else if (cb) {
        cb(null, false);
    }
}

function deleteFolderRecursive(path) {
    if (fs.existsSync(path)) {
        fs.readdirSync(path).forEach(function (file, index) {
            var curPath = path + "/" + file;

            console.log("[Profiles] Delete: " + curPath);
            if (fs.lstatSync(curPath).isDirectory()) { // recurse
                deleteFolderRecursive(curPath);
            } else { // delete file
                fs.unlinkSync(curPath);
            }
        });
    }
}

function resetKeyValStore(path, dbname, useDatabase, cb) {
    if (useDatabase) {
        // clean up the database we created previously
        nano.db.destroy(dbname, function () {
            nano.db.create(dbname, function () {
                console.log("[Profiles] Reset keyValStore database");
                if (cb) {
                    cb(null, true);
                }
            });
        });
    } else if (fs.existsSync(path)) {
        deleteFolderRecursive(path);
        console.log("[Profiles] Reset keyValStore database");
        if (cb) {
            cb(null, true);
        }
    }
}

function resetProfile(cb) {
    if (nano) {
        // clean up the database we created previously
        nano.db.destroy(profiles_creds.database, function () {
            nano.db.create(profiles_creds.database, function () {
                console.log("[Profiles] Reset profiles database");
                if (cb) {
                    cb(null, true);
                }
            });
        });
    } else if (fs.existsSync(profileDir)) {
        deleteFolderRecursive(profileDir);
        console.log("[Profiles] Reset profiles database");
        if (cb) {
            cb(null, true);
        }
    }
}

function changeProfilePassword(username, password, cb) {
    getUserProfileInfo(username, function (userData) {
        if (userData != null) {
            encoding.hashPassword(password, function (errHash, hashResult) {
                if (!errHash) {
                    userData.password = hashResult;
                    saveProfile(username, userData, cb);
                } else {
                    cb(errHash);
                }
            });
        } else {
            cb("No User Data");
        }
    });
}

function updateProfileProperties(username, obj, cb) {
    getUserProfileInfo(username, function (userData) {
        if (userData != null) {
            userData.customProperties = obj;
            saveProfile(username, userData, cb);
        } else {
            cb("No User Data");
        }
    });
}

function checkLogin(checkVariable) {
    if (checkVariable) {
        if (!checkVariable || checkVariable == '') {
            console.log('[Profiles] No session is logged in');
            return false;
        }
    } else {
        console.log('[Profiles] No session is logged in');
        return false;
    }

    console.log("[Profiles] Session found: " + checkVariable);

    return true;
}

function assignDBProfile(username, password, enrollmentPassword, customProperties, cb) {
    var userPublicKey;
    var userPrivateKey;

    if (createUserRSAKeys) {
        var NodeRSA = require('node-rsa');
        var key = new NodeRSA({ b: 512 });

        console.log("[Profiles] Generating keys for " + username + "...");
        key.generateKeyPair();

        userPublicKey = new Buffer(key.exportKey('pkcs8-public-pem')).toString('hex');
        userPrivateKey = new Buffer(key.exportKey('pkcs8-private-pem')).toString('hex');
    }

    if (!username) {
        throw "username is undefined";
    }

    if (!password) {
        throw "password is undefined";
    }

    if (!enrollmentPassword) {
        throw "enrollmentPassword is undefined";
    }

    encoding.hashPassword(password, function (errHash, hashResult) {
        saveProfile(username, { username: username.trim(), password: hashResult.toString('hex'), publicKey: userPublicKey, privateKey: userPrivateKey, tagMemberName: username.trim(), tagSecret: enrollmentPassword.trim(), customProperties: customProperties }, function (err) {
            console.log("[Profiles] Created new account with the database or file-based storage:" + username);
            if (cb) {
                cb(username, password, { public: userPublicKey, private: userPrivateKey }, false, err || errHash);
            }
        });
    });
}

function registerProfile(username, account, affiliation, userRoles, availableRoles, newUserAvailableRoles, customProperties, cb) {
    var requestRegister = {
        enrollmentID: username,
        account: account,
        affiliation: affiliation,
        roles: userRoles,
        registrar: {
            roles: availableRoles,
            delegateRoles: newUserAvailableRoles,
        },
    };

    if (!username) {
        throw "username is undefined";
    }

    if (!account) {
        throw "account is undefined";
    }

    if (!affiliation) {
        throw "affiliation is undefined";
    }

    currChain.getMember(username.trim(), function (errCA, memberCA) {
        var exists = false;

        if (memberCA) {
            exists = memberCA.isRegistered() || memberCA.isEnrolled();
        }

        if (exists) {
            cb(username, "", { public: "", private: "" }, exists, null);
            return
        }

        currChain.register(requestRegister, function (err, enrollmentPassword) {
            if (!err) {
                console.log("[Profiles] Created new account with the sdk:" + username);
                assignDBProfile(username, enrollmentPassword, enrollmentPassword, customProperties, cb);
            } else {
                console.log("[Profiles] Error: Creating new account with the sdk:" + username + " Reason:" + err);
                cb(username, "", { public: "", private: "" }, exists, err);
            }
        });
    });
}

function registerProfileWithPass(username, password, account, affiliation, userRoles, availableRoles, newUserAvailableRoles, customProperties, cb) {
    var requestRegister = {
        enrollmentID: username,
        account: account,
        affiliation: affiliation,
        roles: userRoles,
        registrar: {
            roles: availableRoles,
            delegateRoles: newUserAvailableRoles,
        },
    };

    if (!username) {
        throw "username is undefined";
    }

    if (!password) {
        throw "password is undefined";
    }

    if (!account) {
        throw "account is undefined";
    }

    if (!affiliation) {
        throw "affiliation is undefined";
    }

    currChain.getMember(username.trim(), function (errCA, memberCA) {
        var exists = false;

        if (memberCA) {
            exists = memberCA.isRegistered() || memberCA.isEnrolled();
        }

        if (exists) {
            cb(username, "", { public: "", private: "" }, exists, null);
            return
        }

        currChain.register(requestRegister, function (err, enrollmentPassword) {
            if (!err) {
                console.log("[Profiles] Created new account with the sdk:" + username);
                assignDBProfile(username, password, enrollmentPassword, customProperties, cb);
            } else {
                console.log("[Profiles] Error: Creating new account with the sdk:" + username + " Reason:" + err);
                cb(username, "", { public: "", private: "" }, exists, err);
            }
        });
    });
}

function registerProfileByUser(registrarUsername, username, password, account, affiliation, userRoles, availableRoles, newUserAvailableRoles, customProperties, cb) {
    var requestRegister = {
        enrollmentID: username,
        account: account,
        affiliation: affiliation,
        roles: userRoles,
        registrar: {
            roles: availableRoles,
            delegateRoles: newUserAvailableRoles,
        },
    };

    if (!username) {
        throw "username is undefined";
    }

    if (!account) {
        throw "account is undefined";
    }

    if (!affiliation) {
        throw "affiliation is undefined";
    }

    currChain.getMember(registrarUsername.trim(), function (errRegistrar, memberRegistrar) {
        if (!errRegistrar) {
            currChain.getMember(username.trim(), function (errCA, memberCA) {
                var exists = false;

                if (memberCA) {
                    exists = memberCA.isRegistered() || memberCA.isEnrolled();
                }

                if (exists) {
                    cb(username, "", { public: "", private: "" }, exists, null);
                    return
                }

                currChain.getMemberServices().register(requestRegister, memberRegistrar, function (err, enrollmentPassword) {
                    if (!err) {
                        console.log("[Profiles] Created new account with the sdk:" + username);
                        assignDBProfile(username, enrollmentPassword, enrollmentPassword, customProperties, cb);
                    } else {
                        console.log("[Profiles] Error: Creating new account with the sdk:" + username + " Reason:" + err);
                        cb(username, "", { public: "", private: "" }, exists, err);
                    }
                });
            });
        } else {
            console.log("[Profiles] Error: Creating new account with the sdk:" + username + " Reason:" + errRegistrar);
            cb(username, "", { public: "", private: "" }, false, errRegistrar);
        }
    });
}

function registerProfileByUserWithPass(registrarUsername, username, password, account, affiliation, userRoles, availableRoles, newUserAvailableRoles, customProperties, cb) {
    var requestRegister = {
        enrollmentID: username,
        account: account,
        affiliation: affiliation,
        roles: userRoles,
        registrar: {
            roles: availableRoles,
            delegateRoles: newUserAvailableRoles,
        },
    };

    if (!username) {
        throw "username is undefined";
    }

    if (!password) {
        throw "password is undefined";
    }

    if (!account) {
        throw "account is undefined";
    }

    if (!affiliation) {
        throw "affiliation is undefined";
    }

    currChain.getMember(registrarUsername.trim(), function (errRegistrar, memberRegistrar) {
        if (!errRegistrar) {
            currChain.getMember(username.trim(), function (errCA, memberCA) {
                var exists = false;

                if (memberCA) {
                    exists = memberCA.isRegistered() || memberCA.isEnrolled();
                }

                if (exists) {
                    cb(username, "", { public: "", private: "" }, exists, null);
                    return
                }

                currChain.getMemberServices().register(requestRegister, memberRegistrar, function (err, enrollmentPassword) {
                    if (!err) {
                        console.log("[Profiles] Created new account with the sdk:" + username);
                        assignDBProfile(username, password, enrollmentPassword, customProperties, cb);
                    } else {
                        console.log("[Profiles] Error: Creating new account with the sdk:" + username + " Reason:" + err);
                        cb(username, "", { public: "", private: "" }, exists, err);
                    }
                });
            });
        } else {
            console.log("[Profiles] Error: Creating new account with the sdk:" + username + " Reason:" + errRegistrar);
            cb(username, "", { public: "", private: "" }, false, errRegistrar);
        }
    });
}

function loginProfile(usernameFromReq, passwordFromReq, cb) {
    if (!usernameFromReq) {
        throw "username is undefined";
    }

    if (!passwordFromReq) {
        throw "password is undefined";
    }

    getUserProfileInfo(usernameFromReq, function (loginData) {
        console.log("[Profiles] Checking login information...");
        if (loginData) {
            if (loginData.username && loginData.password) {
                console.log("[Profiles] Logging in: " + loginData.username + "...");
                encoding.verifyPassword(passwordFromReq.trim(), new Buffer(loginData.password, 'hex'), function (err, valid) {
                    if (valid) {
                        console.log("[Profiles] Enrolling user: " + loginData.username + "...");
                        currChain.enroll(loginData.tagMemberName, loginData.tagSecret, function (err, member) {
                            if (member) {
                                console.log("[Profiles] Enrolled user: " + loginData.username);
                                cb(true, false, { username: member.getName(), password: "", keys: { public: loginData.publicKey, private: loginData.privateKey }, customProperties: loginData.customProperties }, member.isRegistered());
                            } else {
                                console.log("[Profiles] [ERROR] Unable to enroll: " + loginData.username);
                                cb(false, false, null, false);
                            }
                        });
                    } else {
                        console.log("[Profiles] [ERROR] Unable to login user: " + loginData.username);
                        cb(false, false, null, true);
                    }
                });
            } else {
                cb(false, false, null, false);
            }
        } else {
            console.log("[Profiles] Checking if the user is enrolled...");
            currChain.getMember(usernameFromReq.trim(), function (errCA, memberCA) {
                if (errCA == null && memberCA) {
                    if (!memberCA.isEnrolled()) {
                        console.log("[Profiles] Enrolling user: " + memberCA.getName() + "...");
                        currChain.enroll(usernameFromReq.trim(), passwordFromReq.trim(), function (err, member) {
                            if (!err) {
                                if (member) {
                                    console.log("[Profiles] Enrolled user: " + member.getName());

                                    assignDBProfile(member.getName(), passwordFromReq.trim(), passwordFromReq.trim(), defaultCustomUserProperties, function (profileUser, profilePass, profileKeys) {
                                        cb(true, true, { username: profileUser, password: "", keys: profileKeys, customProperties: defaultCustomUserProperties }, true);
                                    });
                                } else {
                                    console.log("[Profiles] [ERROR] Unable to enroll: " + usernameFromReq.trim());
                                    cb(false, true, null, true);
                                }
                            } else {
                                console.log("[Profiles] [ERROR] Unable to enroll: " + usernameFromReq.trim());
                                cb(false, true, null, true);
                            }
                        });
                    } else {
                        console.log("[Profiles] Error: User is already enrolled");
                        cb(false, true, null, true);
                    }
                } else {
                    console.log("[Profiles] [ERROR] Unable to find user: " + usernameFromReq.trim());
                    cb(false, true, null, false);
                }
            });
        }
    });
}

function loadProfile(username, cb) {
    if (db) {
        get_user_doc(username.toLowerCase(), function (err, memberdoc) {
            if (err == null) {
                if (cacheUserProfile) {
                    pushUserProfileToCache(memberdoc);
                    console.log("[Profiles] Number of profiles loaded: " + profilesStorage.length);
                }

                console.log("[Profiles] User found in the database: " + username);
                cb(memberdoc);
            } else {
                console.log("[Profiles] User not found in the database: " + username);
                cb(null);
            }
        });
    } else {
        var fs = require("fs");
        var content = fs.readFileSync(profileDir + username.toLowerCase().trim() + ".json");

        if (encoding.isJsonString(content)) {
            var converted = JSON.parse(content);

            if (cacheUserProfile) {
                pushUserProfileToCache(converted.certs);
                console.log("[Profiles] Number of profiles loaded: " + profilesStorage.length);
            }

            cb(converted.certs);
        } else {
            console.log("[Profiles] Error: failed to load profile");
            cb(null);
        }
    }
}

function saveProfile(username, object, cb) {
    if (db) {
        get_user_doc(username, function (currerr, currmemberdoc) {
            var memberdoc = object;
            memberdoc._id = username.toLowerCase();

            if (currerr != null) {
                insert_doc(memberdoc, function (err, userdoc) {
                    if (err) {
                        console.log("[Profiles] ", err);
                    } else {
                        console.log("[Profiles] " + username + "`s profile was saved!");
                    }

                    if (cb) {
                        cb(err);
                    }
                });
            } else {
                memberdoc._rev = currmemberdoc._rev;
                insert_doc(memberdoc, function (err, userdoc) {
                    if (err) {
                        console.log("[Profiles] ", err);
                    } else {
                        console.log("[Profiles] " + username + "`s profile was saved!");
                    }

                    if (cb) {
                        cb(err);
                    }
                });
            }
        });
    } else {
        fs.writeFile(profileDir + username.toLowerCase().trim() + ".json", JSON.stringify({ certs: object }), function (err) {
            if (cb) {
                cb("[Profiles] ", err);
            }
        });

        console.log("[Profiles] " + username + "`s profile was saved!");
        if (cb) {
            cb(null);
        }
    }

    var temp = [];
    for (var index = 0; index < profilesStorage.length; index++) {
        if (profilesStorage[index]) {
            if (username.toLowerCase() != profilesStorage[index].username.toLowerCase()) {
                temp.push(profilesStorage[index]);
            }
        }
    }

    profilesStorage = temp;
}

function getUserProfileInfo(username, ccb) {
    if (!username) {
        throw "username is undefined";
    }

    if (cacheUserProfile) {
        for (var i = 0; i < profilesStorage.length; i++) {
            if (profilesStorage[i].username.toLowerCase() == username.toLowerCase().trim()) {
                console.log("[Profiles] User found in cache: " + username);
                ccb(profilesStorage[i]);
                return;
            }
        }
    }

    if (db) {
        loadProfile(username, function (userData) {
            ccb(userData);
        });
    } else {
        if (fs.existsSync(profileDir + username.trim() + ".json")) {
            console.log("[Profiles] User found on disk: " + username);
            loadProfile(username, function (userData) {
                ccb(userData);
            });
        } else {
            console.log("[Profiles] User not found on disk: " + username);
            ccb(null);
        }
    }
}

function pushUserProfileToCache(memberObject) {
    if (!memberObject) {
        throw "memberObject is undefined";
    }

    profilesStorage.push(memberObject);
}

function setDefaultUserProperties(object) {
    if (!object) {
        throw "user properties is undefined";
    }

    defaultCustomUserProperties = object;
}

function setCacheUserProfile(value) {
    cacheUserProfile = value;
}

function setCreateUserKeys(value) {
    createUserRSAKeys = value;
}

function signRequest(privateKey, args) {
    const sign = crypto.createSign('RSA-SHA256');
    var signString = "";

    for (i = 0; i < args.length; i++) {
        signString = signString + args[i].trim();
    }

    sign.update(signString);

    return sign.sign(privateKey).toString('hex');
}

function verifyRequest(publicKey, signature, args) {
    const verify = crypto.createVerify('RSA-SHA256');
    var verifyString = "";

    for (i = 0; i < args.length; i++) {
        verifyString = verifyString + args[i].trim();
    }

    verify.update(verifyString);

    return verify.verify(publicKey, signature).toString('hex');
}

function get_user_doc(username, cb) {
    db.get(username, { revs_info: true }, function (err, body) {		//doc_name, query parameters, callback
        if (cb) {
            if (!err && body) cb(null, body);
            else if (err && err.statusCode) cb(err.statusCode, { error: err.error, reason: err.reason });
            else cb(500, { error: err, reason: 'unknown!' });
        }
    });
}

function insert_doc(doc, cb) {
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
}

module.exports.chainConfig = function (chain, chainId) {
    //do config
    currChain = chain;
    currChainId = chainId;

    console.log("[Profiles] Connected to HFC SDK");
};

module.exports.setupStorage = function (dir, credentials, useDatabase, cb) {
    if (!profiles_creds || profiles_creds == {}) {
        useDatabase = false;
    }

    profileDir = dir + '/';

    if (useDatabase) {
        console.log("[Profiles] Using a cloudant database");
        profiles_creds = {
            host: credentials.host,
            port: credentials.port,
            username: credentials.username,
            password: credentials.password,
            database: credentials.database,
        };

        setupConn(cb);
    } else {
        console.log("[Profiles] Using a file-based database");

        if (!fs.existsSync(dir)) {
            fs.mkdirSync(dir);
        }

        if (cb) {
            cb(null, true);
        }
    }
};

module.exports.assignDBProfile = assignDBProfile;
module.exports.registerProfile = registerProfile;
module.exports.registerProfileWithPass = registerProfileWithPass;
module.exports.registerProfileByUser = registerProfileByUser;
module.exports.registerProfileByUserWithPass = registerProfileByUserWithPass;
module.exports.resetProfile = resetProfile;
module.exports.resetKeyValStore = resetKeyValStore;
module.exports.getUserProfileInfo = getUserProfileInfo;
module.exports.changeProfilePassword = changeProfilePassword;
module.exports.updateProfileProperties = updateProfileProperties;
module.exports.setDefaultUserProperties = setDefaultUserProperties;
module.exports.setCacheUserProfile = setCacheUserProfile;
module.exports.setCreateUserKeys = setCreateUserKeys;
module.exports.signRequest = signRequest;
module.exports.verifyRequest = verifyRequest;
module.exports.checkLogin = checkLogin;
module.exports.loginProfile = loginProfile;