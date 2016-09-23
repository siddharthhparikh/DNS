var express = require('express');
var router = express.Router();

/* GET home page. */
router.get('/', function(req, res, next) {
  res.render('index', { title: 'DNS Chain' });
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

router.route('/mydomains').get(function(req, res) {
  res.render('managedomains', { title: 'My Domains' });
});

router.route('/mytrades').get(function(req, res) {
  res.render('managetrades', { title: 'My Transfers'});
});

router.route('/search').get(function(req, res) {
  res.render('search', { title: 'DNS Search'});
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

module.exports = router;
