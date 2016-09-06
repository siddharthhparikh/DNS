
// send mail with defined transport object
module.exports.email = function (email, creds, cb) {

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
    if (creds == 'declined') {
        var mailOptions = {
            from: '"Siddharth Parikh" <siddharthparikh1993@gmail.com>', // sender address
            to: email, // list of receivers
            subject: '[Confidential] Vote Chain Account Request Update', // Subject line
            text: 'your account request has been declined',
            html: 'your account request has been declined'

        };
    } else {
        var mailOptions = {
            from: '"Siddharth Parikh" <siddharthparikh1993@gmail.com>', // sender address
            to: email, // list of receivers
            subject: '[Confidential] Vote Chain Password', // Subject line
            text: 'username: ' + creds.id + '\npassword: ' + creds.secret, // plaintext body
            html: 'username: ' + creds.id + '\npassword: ' + creds.secret // html body

        };
    }
    console.log("Mail Options:")
    console.log(mailOptions);
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