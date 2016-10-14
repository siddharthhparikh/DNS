
// send mail with defined transport object
module.exports.email = function (email, keys, cb) {

    var nodemailer = require('nodemailer');

    // create reusable transporter object using the default SMTP transport
    var transporter = nodemailer.createTransport("SMTP", {
        service: "Gmail",
        auth: {
            user: "siddharthparikh1993@gmail.com",
            pass: "siddharth24"
        }
    });
    console.log("email = " + email);
    // setup e-mail data with unicode symbols
    var mailOptions = {
        from: '"Siddharth Parikh" <siddharthparikh1993@gmail.com>', // sender address
        to: email, // list of receivers
        subject: '[Confidential] Please save your public and private key', // Subject line
        text: "pubkey:\n" + keys.public + '\nPrivate Key:\n' + keys.private, // plaintext body
        html: "pubkey:<br></br><br></br>" + keys.public + '<br></br><br></br>Private Key:<br></br><br></br>' + keys.private // html body
    };
    transporter.sendMail(mailOptions, function (error, info) {
        if (error) {
            console.log(error);
            return cb(error);
            //return console.log(error);
        }
        //console.log('Message sent: ' + info.response);
        return cb(null);
    });
}