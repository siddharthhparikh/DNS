/**
 * @author Ethan Coeytaux
 * 
 * provides functions for interacting with chaincode SDK
 */
    //set the gopath to current dir and place chaincode inside src folder

var hfc = require('hfc');
var fs = require('fs');

var exports = module.exports;

// Create a client chain
var chaincodeName = 'voting_chaincode'
var chain = hfc.newChain("voting");
chain.setDeployWaitTime(300);
var chaincodeID = null;


// Configure the KeyValStore which is used to store sensitive keys
// as so it is important to secure this storage
chain.setKeyValStore(hfc.newFileKeyValStore('/tmp/keyValStore'));

var peerURLs = [];
var caURL = null;
var users = null;
var user_manager = require("./users")
var registrar = null; //user used to register other users and deploy chaincode
var peerURLs = [];
var caURL = null;
var users = null;
var registrar = null; //user used to register other users and deploy chaincode
var peerHosts = [];
var WebAppAdmin;
//hard-coded the peers and CA addresses.
//added for reading configs from file
try {
    var manual = JSON.parse(fs.readFileSync('mycreds.json', 'utf8'));
    var peers = manual.credentials.peers;
    for (var i in peers) {
        peerURLs.push("grpcs://" + peers[i].discovery_host + ":" + peers[i].discovery_port);
        peerHosts.push("" + peers[i].discovery_host);
    }
    var ca = manual.credentials.ca;
    for (var i in ca) {
        caURL = "grpcs://" + ca[i].url;
    }
    console.log('loading hardcoded peers');
    var users = null;																			//users are only found if security is on
    if (manual.credentials.users) users = manual.credentials.users;
    console.log('loading hardcoded users');
}
catch (e) {
    console.log('Error - could not find hardcoded peers/users, this is okay if running in bluemix');
}

if (process.env.VCAP_SERVICES) {
    //load from vcap, search for service, 1 of the 3 should be found...
    var servicesObject = JSON.parse(process.env.VCAP_SERVICES);
    for (var i in servicesObject) {
        if (i.indexOf('ibm-blockchain') >= 0) {											//looks close enough
            if (servicesObject[i][0].credentials.error) {
                console.log('!\n!\n! Error from Bluemix: \n', servicesObject[i][0].credentials.error, '!\n!\n');
                peers = null;
                users = null;
                process.error = { type: 'network', msg: 'Due to overwhelming demand the IBM Blockchain Network service is at maximum capacity.  Please try recreating this service at a later date.' };
            }
            if (servicesObject[i][0].credentials && servicesObject[i][0].credentials.peers) {
                console.log('overwritting peers, loading from a vcap service: ', i);
                peers = servicesObject[i][0].credentials.peers;
                peerURLs = [];
                peerHosts = [];
                for (var j in peers) {
                    peerURLs.push("grpcs://" + peers[j].discovery_host + ":" + peers[j].discovery_port);
                    peerHosts.push("" + peers[j].discovery_host);
                }
                if (servicesObject[i][0].credentials.ca) {
                    console.log('overwritting ca, loading from a vcap service: ', i);
                    ca = servicesObject[i][0].credentials.ca;
                    for (var z in ca) {
                        caURL = "grpcs://" + ca[z].discovery_host + ":" + ca[z].discovery_port;
                    }
                    if (servicesObject[i][0].credentials.users) {
                        console.log('overwritting users, loading from a vcap service: ', i);
                        users = servicesObject[i][0].credentials.users;
                        //TODO extract registrar from users once user list has been updated to new SDK
                    }
                    else users = null;													//no security	
                }
                else ca = null;
                break;
            }
        }
    }
}

var pwd = "";
for (var z in users) {
    if (users[z].username == "WebAppAdmin") {
        pwd = users[z].secret;
    }
}

if (fs.existsSync("us.blockchain.ibm.com.cert")) {
    var pem = fs.readFileSync('us.blockchain.ibm.com.cert');

    chain.setECDSAModeForGRPC(true);

    console.log('loading hardcoding users and certificate authority...')

    // Set the URL for member services
    console.log('adding ca: \'' + caURL + '\'');
    chain.setMemberServicesUrl(caURL, { pem: pem });

    // Add all peers' URL
    for (var i in peerURLs) {
        console.log('adding peer: \'' + peerURLs[i] + '\'');
        chain.addPeer(peerURLs[i], { pem: pem });
    }

    chain.getMember("WebAppAdmin", function (err, WebAppAdmin) {
        if (err) {
            console.log("Failed to get WebAppAdmin member " + " ---> " + err);
        } else {
            console.log("Successfully got WebAppAdmin member" + " ---> " /*+ JSON.stringify(crypto)*/);

            // Enroll the WebAppAdmin member with the certificate authority using
            // the one time password hard coded inside the membersrvc.yaml.
            WebAppAdmin.enroll(pwd, function (err, crypto) {
                if (err) return console.log('Error: failed to enroll user: %s', err);
                console.log('successfully enrolled user \'%s\'!', "WebAppAdmin");
                path = chain.getKeyValStore().dir + "/member." + WebAppAdmin.getName();
                registrar = WebAppAdmin;
                fs.exists(path, function (exists) {
                    if (exists) {
                        console.log("Successfully stored client token for" + " ---> " + WebAppAdmin.getName());
                    } else {
                        console.log("Failed to store client token for " + WebAppAdmin.getName() + " ---> " + err);
                    }
                });

                chain.setRegistrar(WebAppAdmin);
                console.log('enrolling user \'%s\' with secret \'%s\' as registrar...', "WebAppAdmin", pwd);
                exports.deploy('', ['ready!'], function (chaincodeID) {
                    user_manager.setup(chaincodeID, chain, cb_deployed);
                });

            });
        };
    });
} else {
    console.log('[ERROR] us.blockchain.ibm.com.cert not found')
}

function cb_deployed() {

}

///////////////////////////////
// CHAINCODE SDK HELPER FUNCTIONS
///////////////////////////////

//deploys chaincode (cb in form of cb(err))
exports.deploy = function (path, args, cb) {
    if (registrar == null) {
        console.log('ERROR: attempted to deploy chaincode without initializing registrar...');
        return;
    }

    var deployRequest = {
        args: args,
        //chaincodeID: chaincodeName,
        fcn: 'init',
        chaincodePath: path,
        certificatePath: "/certs/blockchain-cert.pem"
    }
    console.log('deploying chaincode from path %s', deployRequest.chaincodePath)
    var transactionContext = registrar.deploy(deployRequest);

    transactionContext.on('submitted', function (results) {
        console.log('chaincode submitted successfully!');
        console.log('chaincode-ID: %s', results.chaincodeID);

        chaincodeID = results.chaincodeID;
        //chaincode has been deployed

        if (cb) cb(chaincodeID);
    });

    transactionContext.on('error', function (err) {
        console.log('Error deploying chaincode: %s', err.msg);
        console.log('App will fail without chaincode, sorry!');

        //chaincode has errored

        cb(err);
    });
}

//invokes function on chaincode (cb in form of cb(err, result))
exports.invoke = function (fcn, args, cb) {
    if (chaincodeID == "" || chaincodeID == null) {
        return new Error("No chaincode ID implies chaincode has not yet deployed");
    }

    var invokeRequest = {
        fcn: fcn,
        args: args,
        chaincodeID: chaincodeID
    }

    var transactionContext = registrar.invoke(invokeRequest);

    transactionContext.on('complete', function (results) {
        if (cb) {
            if (results.result) {
                console.log("In invoke results on complete")
                console.log(results)
                console.log(results.result)
                cb(null, results.result)
            } else {
                cb(new Error("no data retrieved from invoke"), null);
            }
        }
    });

    transactionContext.on('error', function (err) {
        if (cb) {
            cb(err, null);
        }
    });
}

//queries on chaincode (cb in form of cb(err, result))
exports.query = function (fcn, args, expectJSON, cb) {
    if (chaincodeID == "" || chaincodeID == null) {
        return new Error("No chaincode ID implies chaincode has not yet deployed");
    }

    if (typeof expectJSON === 'function') { //only 3 parameters passed, expectJSON defaults to true
        cb = expectJSON;
        expectJSON = true;
    }

    var queryRequest = {
        fcn: fcn,
        args: args,
        chaincodeID: chaincodeID
    }

    var transactionContext = registrar.query(queryRequest);

    transactionContext.on('complete', function (results) {
        if (cb) {
            console.log("query completed with results:")
            console.log(results)
            if (results.result) { //is result is not null
                //parse data from buffer to json
                var data = String.fromCharCode.apply(String, results.result);
                if (expectJSON) {
                    if (data.length > 0) cb(null, JSON.parse(data));
                    else cb(null, null);
                } else {
                    cb(null, data)
                }
            } else {
                cb(new Error("no data retrieved from query"), null);
            }
        }
    });

    transactionContext.on('error', function (err) {
        if (cb) {
            console.log("query completed with error:")
            console.log(err)
            cb(err, null);
        }
    });
}

module.exports.registerAndEnroll = function (username, role, cb) {
    return user_manager.registerUser(username, role, cb);
}

module.exports.login = function (username, secret, cb) {
    console.log("I am inside blockchainsdk.js login function")
    return user_manager.login(username, secret, cb);
}