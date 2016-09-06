/*******************************************************************************
 * Copyright (c) 2016 IBM Corp.
 *
 * All rights reserved. 
 *
 * Contributors:
 *   Justin E. Ervin - Initial implementation
 *******************************************************************************/

var encoding = require('./encoding');
var purchasedAssetRate = "0.00273972603"; // 1 / 365
var renewAssetRate = "0.00273972603";  // 1 / 365
var useAssetLengthForPurchasedCost = "false";
var useAssetLengthForRenewCost = "false";
var doesRenewAssetCost = "false";
var startingAccountMoney = "100000.00";
var myProfiles;

function sendMsg(ws, json) {
  if (ws) {
    try {
      console.log("[WS] SENT: " + JSON.stringify(json));
      ws.send(JSON.stringify(json));
    }
    catch (e) {
      console.log('[WS] error ws', e);
    }
  }
}

module.exports.sendMsg = sendMsg;

function broadcast(wss, json) {
  wss.clients.forEach(function each(client) {
    try {
      client.send(JSON.stringify(json));
    }
    catch (e) {
      console.log('[WS] error broadcast ws', e);
    }
  });
}

function process_msg(chain, chainId, ws, wss, data) {
  if (data.type == "query" && chainId && chain) {
    if (data.fcn == "query_stats") {
      sendQuery(chain, chainId, ws, wss, data.fcn, [], false);
    } else if (data.fcn == "user_asset_records") {
      sendQuery(chain, chainId, ws, wss, data.fcn, data.args, false);
    } else if (data.fcn == "user_trade_records") {
      sendQuery(chain, chainId, ws, wss, data.fcn, data.args, false);
    } else if (data.fcn == "asset_record") {
      sendQuery(chain, chainId, ws, wss, data.fcn, data.args, false);
    } else if (data.fcn == "dns_record") {
      sendQuery(chain, chainId, ws, wss, data.fcn, data.args, false);
    } else if (data.fcn == "profile_record") {
      sendQuery(chain, chainId, ws, wss, data.fcn, data.args, false);
    }
  } else if (data.type == "invoke" && chainId && chain) {
    if (data.fcn == "profile_update") {
      if (data.args.length >= 2) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [enrollment.publicKey, data.args[1]]);
            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [enrollment.publicKey, signature, data.args[1]], function () {
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_init") {
      if (data.args.length >= 6) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], enrollment.publicKey, data.args[2], data.args[3], data.args[4], data.args[5], useAssetLengthForPurchasedCost, purchasedAssetRate]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature, enrollment.publicKey, data.args[2], data.args[3], data.args[4], data.args[5], useAssetLengthForPurchasedCost, purchasedAssetRate], function () {
              sendQuery(chain, chainId, ws, wss, "query_stats", [], true);
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_renew") {
      if (data.args.length >= 3) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], data.args[2], useAssetLengthForRenewCost, doesRenewAssetCost, renewAssetRate]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature, data.args[2], useAssetLengthForRenewCost, doesRenewAssetCost, renewAssetRate], function () {
              sendQuery(chain, chainId, ws, wss, "user_asset_records", [data.args[0]], false);
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_transfer_init") {
      if (data.args.length >= 6) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], data.args[2], data.args[3], data.args[4], data.args[5]]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature, data.args[2], data.args[3], data.args[4], data.args[5]], function () {
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_transfer_accept") {
      if (data.args.length >= 6) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var assetSignature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], enrollment.publicKey, data.args[2], data.args[3], data.args[4], data.args[5], useAssetLengthForPurchasedCost, purchasedAssetRate]);
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], enrollment.publicKey, assetSignature]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature, enrollment.publicKey, assetSignature], function () {
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_transfer_decline") {
      if (data.args.length >= 2) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], enrollment.publicKey]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature, enrollment.publicKey], function () {
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_transfer_make_offer") {
      if (data.args.length >= 3) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], enrollment.publicKey, data.args[2]]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature, enrollment.publicKey, data.args[2]], function () {
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_transfer_cancel") {
      if (data.args.length >= 2) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1]]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature], function () {
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "asset_delete") {
      if (data.args.length >= 2) {
        myProfiles.getUserProfileInfo(data.args[0], function (enrollment) {
          if (typeof enrollment !== 'undefined') {
            var signature = myProfiles.signRequest(new Buffer(enrollment.privateKey, 'hex').toString(), [data.args[1], "true", "1.0"]);

            sendUserInvoke(chain, chainId, data.args[0], ws, wss, data.fcn, [data.args[1], signature, "true", "1.0"], function () {
              sendQuery(chain, chainId, ws, wss, "user_asset_records", [data.args[0]], false);
            });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    }
  } else if (data.type == "other") {
    if (data.fcn == "register" && chainId && chain) {
      if (data.args.length >= 1) {
        console.log("Creating new account:" + data.args[0]);
        sendMsg(ws, { fcn: "profile_init", message: "Creating new account", messageType: "waiting" });
        myProfiles.registerProfile(data.args[0], "group1", "00001", [], [], [], {}, function (profileUser, profilePass, profileKeys, userExists, registerErr) {
          if (!userExists) {
            if (!registerErr) {
              sendInvoke(chain, chainId, ws, wss, "profile_init", [data.args[0], profileKeys.public, "", startingAccountMoney], function () {
                sendMsg(ws, { fcn: "profile_init", message: "register_completed", username: profileUser, password: profilePass, messageType: "info" });
                sendQuery(chain, chainId, ws, wss, "query_stats", [], true);
              });
            } else {
              sendMsg(ws, { fcn: "profile_init", message: "Error: Failed to register the user!", messageType: "error" });
            }
          } else {
            sendMsg(ws, { fcn: "profile_init", message: "Error: This username is already registered!", messageType: "error" });
          }
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    } else if (data.fcn == "check_chain") {
      if (chainId && chain) {
        sendMsg(ws, { fcn: "check_chain", message: "online", messageType: "info" });
      } else {
        sendMsg(ws, { fcn: "check_chain", message: "offine", messageType: "info" });
      }
    } else if (data.fcn == "request_private_key") {
      if (data.args.length >= 2) {
        myProfiles.getUserProfileInfo(data.args[0], function (userData) {
          encoding.verifyPassword(data.args[1], new Buffer(userData.password, 'hex'), function (err, valid) {
            if (valid) {
              sendMsg(ws, { fcn: "request_private_key", message: userData.privateKey, messageType: "info" });
            } else {
              sendMsg(ws, { fcn: "request_private_key", message: "Error: Incorrect password!", messageType: "error" });
            }
          });
        });
      } else {
        sendMsg(ws, { fcn: data.fcn, message: "Error: Missing inputs", messageType: "error" });
      }
    }
  }
};

module.exports.process_msg = process_msg;

function sendUserInvoke(chain, chainId, username, ws, wss, fcn, args, cb) {
  var requestInvoke = {
    args: args,
    chaincodeID: chainId,
    fcn: fcn
  };

  chain.getMember(username, function (err, member) {
    if (!err) {
      var tx = member.invoke(requestInvoke);

      // Listen for the 'submitted' event
      tx.on('submitted', function (results) {
        console.log("[HFC SDK] submitted invoke: %j", results);
        if (ws != null) {
          sendMsg(ws, { fcn: fcn, message: "submitted" });
        }
      });
      // Listen for the 'complete' event.
      tx.on('complete', function (results) {
        console.log("[HFC SDK] completed invoke: %j", results);
        if (ws != null) {
          sendMsg(ws, { fcn: fcn, message: "complete" });
        }
        if (cb) {
          cb();
        }
      });
      // Listen for the 'error' event.
      tx.on('error', function (err) {
        console.log("[HFC SDK] error on invoke: %j", err);
        if (ws != null) {
          sendMsg(ws, { fcn: fcn, message: "error", error: err });
        }
      });
    }
  });
}

function sendInvoke(chain, chainId, ws, wss, fcn, args, cb) {
  var requestInvoke = {
    args: args,
    chaincodeID: chainId,
    fcn: fcn
  };

  var tx = chain.getRegistrar().invoke(requestInvoke);

  // Listen for the 'submitted' event
  tx.on('submitted', function (results) {
    console.log("[HFC SDK] submitted invoke: %j", results);
    if (ws != null) {
      sendMsg(ws, { fcn: fcn, message: "submitted" });
    }
  });
  // Listen for the 'complete' event.
  tx.on('complete', function (results) {
    console.log("[HFC SDK] completed invoke: %j", results);
    if (ws != null) {
      sendMsg(ws, { fcn: fcn, message: "complete" });
    }

    if (cb) {
      cb();
    }
  });
  // Listen for the 'error' event.
  tx.on('error', function (err) {
    console.log("[HFC SDK] error on invoke: %j", err);
    if (ws != null) {
      sendMsg(ws, { fcn: fcn, message: "error", error: err });
    }
  });
};

function sendQuery(chain, chainId, ws, wss, fcn, args, isbroadcast) {
  var requestQuery = {
    args: args,
    chaincodeID: chainId,
    fcn: fcn
  };

  var tx = chain.getRegistrar().query(requestQuery);

  // Listen for the 'submitted' event
  tx.on('submitted', function (results) {
    console.log("[HFC SDK] submitted query: %j", results);
    if (ws != null) {
      sendMsg(ws, { fcn: fcn, message: "submitted" });
    }
  });
  // Listen for the 'complete' event.
  tx.on('complete', function (results) {
    console.log("[HFC SDK] completed query: %j", results);
    if (isbroadcast) {
      if (wss != null) {
        broadcast(wss, { fcn: fcn, message: results.result });
      }
    } else {
      if (ws != null) {
        sendMsg(ws, { fcn: fcn, message: results.result });
      }
    }
  });
  // Listen for the 'error' event.
  tx.on('error', function (err) {
    console.log("[HFC SDK] error on query: %j", err);
    if (ws != null) {
      sendMsg(ws, { fcn: fcn, message: "error", error: err });
    }
  });
};

module.exports.sendInvoke = sendInvoke;
module.exports.sendQuery = sendQuery;
module.exports.sendUserInvoke = sendUserInvoke;
module.exports.config = function (myprofiles) {
  myProfiles = myprofiles;
};