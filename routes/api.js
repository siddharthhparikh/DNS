/**
 * @author Gennaro Cuomo
 * @author Ethan Coeytaux
 * 
 * Handles all api calls from the client.
 * Interfaces with the chaincode to get client requested information.
 */
var express = require('express');
var router = express.Router();
var session = require('express-session');
var chaincode = require('../libs/blockchainSDK');
var mail = require('../libs/mail');
var cryptico = require('cryptico');
var mkdirp = require('mkdirp');

/* Login in request. */
router.post('/login', function (req, res, next) {
  // Set up the user object for the chaincode.
  var user = req.body;
  // TODO check if the user already exsits in db.

  console.log("[USER]", user);

  var username = user.username;
  var password = user.password;
  var sign = user.sign;
  console.log("inside /login");
  var args = [username, sign, password];
  console.log(args)
  chaincode.query("checkAccount", args, function (err, data) {
    //console.log("[ERROR]", err)
    if (err != null) {
      console.log(err.msg);
      res.end('{"status" : "ERROR: Check server logs"}');
    }
    else {
      console.log(user);
      req.session.name = user.username;
      console.log('Logging in as.....');
      console.log(req.session.name);
      //Send response.
      /*
      if (username.indexOf('manager') > -1) {
        res.end('{"status" : "success", "type": "manager", "message": "ok"}');
      }
      else {
        res.end('{"status" : "success", "type": "user", "message": "ok"}');
      }
      */
    }
  });
});
router.post('/search', function (req, res, next) {
  res.end('{"status" : "success", "type": "manager", "data": "success"}')

});
router.get('/get-account', function (req, res) {
  var args = [];
  args.push(req.session.name);
  chaincode.query('get_account', args, function (err, data) {
    if (data) {
      console.log("[ACCOUNT]", data);
      res.json(data);
    } else {
      res.json('{"status" : "could not retrieve user"}');
    }
  });
})

// Clears all topics on blockchain
// TODO this is just for debugging!
// router.get('/o', function (req, res) {
//   console.log('deleting all topics...');
//   console.log('hope you know what you\'re doing...');
//   chaincode.invoke('clear_all_topics', [], function (err, data) {
//     if (err) {
//       console.log('ERROR: ' + err);
//       res.json('{"status" : "failure"}');
//     } else {
//       console.log('delete of all topics successful!');
//       res.json('{"status" : "success"}');
//     }
//   });
// });

/* Get all voting topics from blockchain */
/*
router.get('/get-topics', function (req, res) {
  var args = [];
  args.push(req.session.name);
  chaincode.query('get_all_topics', args, function (err, data) {
    if (err) console.log('ERROR: ', err);
    else res.json(data);
  });
});
*/
/* Get specific voting topic from blockchain */
/*
router.get('/get-topic', function (req, res) {
  console.log('Getting topic...');
  var args = [];
  args.push(req.query.topicID);
  args.push(req.session.name);
  chaincode.query('get_topic', args, function (err, data) {
    if (err) console.log('ERROR: ', err);
    else res.json(data);
  });
});
*/
/* Checks the validity of the given topic */
/*
router.get('/topic-check', function (req, res, next) {
  // Get the topic id from the post
  var args = [];
  args.push(req.query.topicID);
  args.push(req.session.name);
  chaincode.query('get_topic', args, function (err, data) {
    if (err) {
      res.json('{"status" : "failure"}');
    } else {
      res.json('{"status" : "success"}');
    }
  });
});
*/
/* Create a new voting topic */
/*
router.post('/create', function (req, res, next) {
  var newTopic = req.body;

  // Set the issuer to the current active user,
  newTopic.issuer = req.session.name;

  console.log('New topic: \n ' + JSON.stringify(newTopic));
  // Add topic object to database.

  var args = [JSON.stringify(newTopic)];
  console.log("in create before running issue topic args are: " + args);
  chaincode.invoke('issue_topic', args, function (err, results) {
    if (err) console.log(err);
    else res.json('{"status" : "success"}');
  });
});
*/
/* Submit votes from a user */
/*
router.post('/vote-submit', function (req, res, next) {
  req.body.voter = req.session.name;

  chaincode.invoke('cast_vote', JSON.stringify(req.body), function (err, results) {
    res.json('{"status" : "success"}');
  })
});
*/
/* Used to let the client know when the Chaincode is finished loading */
/*
router.get('/load-chain', function (req, res) {
  var args = [];
  args.push('InitState')
  chaincode.query('read', args, false, function (err, results) {
    if (results == 'ready!') {
      res.json('{"status" : "success"}');
    }
  });
});
*/
/* Get request for current user logged in */
/*
router.get('/user', function (req, res) {
  var user = req.session.name;
  console.log('Fetching current user: ' + user);
  var response = { 'user': user };
  res.json(response);
});
*/
/* Regiister a user */

// gen pub priv key pair
var cp = require('child_process')
  , assert = require('assert')
  , fs = require('fs')
;

function genKeys(email, cb) {
  // gen private
  cp.exec('openssl genrsa 2048', function (err, priv, stderr) {
    // tmp file
    mkdirp('./keys/', function (err) {
      var randomfn = './keys/' + email + '.pem';
      fs.writeFileSync(randomfn, priv);
      // gen public
      cp.exec('openssl rsa -in ' + randomfn + ' -pubout', function (err, pub, stderr) {
        // delete tmp file
        //fs.unlinkSync(randomfn);
        // callback
        cb({ public: pub, private: priv });
      });
    });

  });
}
router.post('/register', function (req, res) {
  console.log(req.body);
  genKeys(req.body.email, function (keys) {
    console.log(keys.public)
    chaincode.invoke('createAccount', [req.body.email, "", keys.public, req.body.password], function (err, results) {
      if (err != null) {
        res.json('{"status" : "failure", "Error": err}');
      }
      console.log("\n\n\nrequest account result:")
      console.log(results);
      mail.email(req.body.email, keys, function (err) {
        if (err != null) {
          res.end('{"status" : "failure", "Error": err}');
        }
      });
      res.json('{"status" : "success", "message":"ok"}');
    });
  });
  res.render('keydownload', {account: req.body.email});
});

router.get('/manager', function (req, res) {
  console.log("in /manager")
  console.log(req.session.name)
  chaincode.query('get_account', [req.session.name], function (err, data) {
    //if (req.session.name.indexOf('manager') > -1) {
    if (data && data.privileges) {
      if (data.privileges.indexOf('manager') > -1) {
        chaincode.query('get_open_requests', [], function (err, data) {
          if (err != null) {
            res.json('{"status" : "failure", "Error": err}');
          }
          console.log(data);
          res.json(data);
        });
      } else {
        res.json('{"status" : "failure", "Error": "You dont have access rights to view this page"}');
      }
    }
  });
});

router.post('/approved', function (req, res) {
  console.log("request approved")
  console.log(req.body)
  console.log(req.body.Email)
  var args = [
    "approved",
    req.body.Name,
    req.body.Email,
    req.body.Org,
    req.body.Privileges,
    req.session.name,
    req.body.VoteCount
  ]
  console.log("In approved args")
  console.log(args)
  chaincode.invoke('change_status', args, function (err, data) {
    if (err != null) {
      console.log("error=" + err)
      res.json('{"status" : "failure", "Error": err}');
    }
    chaincode.query('get_UserID', [req.body.Email], function (err, data) {
      if (err != null) {
        res.json('{"status" : "failure", "Error": err}');
      }
      console.log(data.AllAccReq)
      //console.log(bin2String(data.AllAccReq))
      chaincode.registerAndEnroll(data.AllAccReq, "user", function (err, cred) {
        if (err != null) {
          res.json('{"status" : "failure", "Error": err}');
        }
        console.log("\n\n\ncreate account result:")
        console.log(cred);
        mail.email(req.body.Email, cred, function (err) {
          if (err != null) {
            res.json('{"status" : "failure", "Error": err}');
          }
          //res.json('{"status" : "success"}');
        });
      });
    });
  });
});

router.post('/declined', function (req, res) {
  console.log("request declined")
  console.log(req.body)
  console.log(req.body.Email)
  var args = ["declined", req.body.Name, req.body.Email, req.body.Org, req.body.Privileges];
  console.log("Email sent");
  console.log("For changing status ars are: ")
  console.log(args)
  chaincode.invoke('change_status', args, function (data, err) {
    console.log("status changed");
    mail.email(req.body.Email, "declined", function (err) {
      if (err != null) {
        res.json('{"status" : "failure", "Error": err}');
      }
      //res.json('{"status" : "success"}');
    });
  });
});

router.post('/searchd', function (req, res) {
  chaincode.query('getIPAddress', [req.body.domainName], function (data,err) {
    console.log(data)
    console.log(err)
    /*
      Display Buy if we get an error
      Display IP address if IP exist. Ask to place a bid to buy the domain. 
    */
    res.json('{"data" : data, "Error" : err}')
  });
});
module.exports = router;
