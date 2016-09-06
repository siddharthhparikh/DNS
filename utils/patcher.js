var http = require('http');
var fs = require('fs');
var path = require('path');

var download = function (url, dest, cb) {
  var file = fs.createWriteStream(dest);
  var request = http.get(url, function (response) {
    response.pipe(file);
    file.on('finish', function () {
      file.close(cb);  // close() is async, call cb after close completes.
    });
  }).on('error', function (err) { // Handle errors
    fs.unlink(dest); // Delete the file async. (But we don't check the result)
    if (cb) cb(err.message);
  });
};

var fabricUrl = "https://github.com/hyperledger/fabric/archive/v0.5-developer-preview.zip";
var chaincodeUrl = "";
var certificateUrl = "";

download(fabricUrl, path.join(__dirname, 'chaincode'), function (err) {
  
});