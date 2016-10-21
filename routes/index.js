var express = require('express');
var router = express.Router();

/* GET home page. */
router.get('/', function(req, res, next) {
  res.render('main', { title: 'DNS Chain' });
});

router.route('/myaccount').get(function( req, res) {
  res.render('manageaccount', { title: 'My Account' });
});

router.route('/register').get(function(req, res) {
  res.render('register', {title: 'Register'});
});

router.route('/keys').get(function(req, res) {
  res.render('keys', {title: 'Register'});
});

router.post('/keydown', function(req, res, next) {
  var type = req.body.download;
  if(download = 'public') {
    var name = '123123@123.com' + '.pem';
    var path = './keys/' + name;

  } else {
    //TODO private key here
  }
  res.download(path, name, function(err) {
    if(err) {
      console.log('Error downloading: ', err);
    } else {
      console.log('Key downloaded');
    }
  });


});

router.route('/mydomains').get(function(req, res) {
  res.render('managedomains', { title: 'My Domains' });
});

router.route('/mytrades').get(function(req, res) {
  res.render('managetrades', { title: 'My Transfers'});
});

router.route('/search').get(function(req, res) {
  res.render('search', { title: 'DNS Search', search: req.body.lookup});
});

router.route('/transfer').get(function(req, res) {
  //TODO Check if logged in
  res.render('opentrade', { title: 'DNS Transfer'});
});

router.route('/buy').get(function(req, res) {
  //TODO Check if logged in
  res.render('buydomain', {title: 'Purchase'});
});

router.route('/login').get(function(req, res) {
  res.render('login', {title: 'Login'});
});

router.route('/editaccount').get(function(req,res) {
  res.render('editaccount');
});

router.post('/searchd', function(req, res, next) {
  //TODO use search results
  var searchTerm = req.body.lookupd;
  var results = 'Domain results for \"' + searchTerm + '\"';
  res.render('results', {title: 'DNS Search', resulttype: results});
});

router.post('/searchu', function(req, res, next) {
  //TODO use search results
  var searchTerm = req.body.lookupu;
  var results = 'Domain results for \"' + searchTerm + '\"';
  res.render('results', {title: 'DNS Search', resulttype: results});
});



module.exports = router;
