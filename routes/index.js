/*******************************************************************************
 * Copyright (c) 2016 IBM Corp.
 *
 * All rights reserved. 
 *
 * Contributors:
 *   Justin E. Ervin - Initial implementation
 *******************************************************************************/

var express = require('express');
var router = express.Router();
var currChainId;
var currChain;
var currWS;
var currWSS;
var myChain;
var myProfiles;

/* GET home page. */
router.route("/").get(function (req, res) {
  if (myProfiles) {
    if (!myProfiles.checkLogin(req.session.username)) {
      res.render('index', { title: 'Asset Book', username: "" });
    } else {
      res.render('index', { title: 'Asset Book', username: req.session.username });
    }
  } else {
    res.render('index', { title: 'Asset Book', username: "" });
  }
});

router.route("/myaccount").get(function (req, res) {
  if (myProfiles) {
    if (!myProfiles.checkLogin(req.session.username)) {
      res.redirect('/login');
    } else {
      res.render('manageaccount', { title: 'Asset Book', username: req.session.username });
    }
  } else {
    res.redirect('/login');
  }
});

router.route("/myassets").get(function (req, res) {
  if (myProfiles) {
    if (!myProfiles.checkLogin(req.session.username)) {
      res.redirect('/login');
    } else {
      res.render('manageassets', { title: 'Asset Book', username: req.session.username });
    }
  } else {
    res.redirect('/login');
  }
});

router.route("/mytrades").get(function (req, res) {
  if (myProfiles) {
    if (!myProfiles.checkLogin(req.session.username)) {
      res.redirect('/login');
    } else {
      res.render('managetrades', { title: 'Asset Book', username: req.session.username });
    }
  } else {
    res.redirect('/login');
  }
});

router.route("/tradeasset").get(function (req, res) {
  if (myProfiles) {
    if (!myProfiles.checkLogin(req.session.username)) {
      res.redirect('/login');
    } else {
      res.render('opentrade', { title: 'Asset Book', username: req.session.username, assetName: req.query['asset'] });
    }
  } else {
    res.redirect('/login');
  }
});

router.route("/buyasset").get(function (req, res) {
  if (myProfiles) {
    if (!myProfiles.checkLogin(req.session.username)) {
      res.redirect('/login');
    } else {
      res.render('buyasset', { title: 'Asset Book', username: req.session.username });
    }
  } else {
    res.redirect('/login');
  }
});

router.route("/login").get(function (req, res) {
  res.render('login', { title: 'Asset Book' });
});

router.route("/login").post(function (req, res) {
  if (myProfiles) {
    myProfiles.loginProfile(req.body.username, req.body.password, function (loginOk, isUserFromCA, userData, userExists) {
      if (loginOk) {
        if (userData != null && isUserFromCA) {
          myChain.sendInvoke(currChain, currChainId, null, null, "profile_init", [userData.username, userData.keys.public, "", "10000.00"], function () {
            myChain.sendQuery(currChain, currChainId, currWS, currWSS, "query_stats", [], true);
          });
        }

        req.session.username = userData.username;

        res.render('processlogin', { results: JSON.stringify({ message: "ok" }) });
      } else {
        res.render('processlogin', { results: JSON.stringify({ message: "error" }) });
      }
      console.log(userExists);
    });
  } else {
    res.render('processlogin', { results: "error" });
  }
});

router.route("/lookup").get(function (req, res) {
  if (myProfiles) {
    if (!myProfiles.checkLogin(req.session.username)) {
      res.render('lookup', { title: 'Asset Book', username: "" });
    } else {
      res.render('lookup', { title: 'Asset Book', username: req.session.username });
    }
  } else {
    res.render('lookup', { title: 'Asset Book', username: "" });
  }
});

router.route("/logout").get(function (req, res) {
  req.session.destroy();
  res.redirect("/login");
});

module.exports = router;
module.exports.config = function (chain, chainId, mychain, myprofiles) {
  //do config
  currChain = chain;
  currChainId = chainId;
  myChain = mychain;
  myProfiles = myprofiles;
};
module.exports.WSconfig = function (ws, wss) {
  currWS = ws;
  currWSS = wss;
};
