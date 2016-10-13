
// send mail with defined transport object
module.exports.email = function (email, cb) {

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
        subject: '[Confidential] Private Key', // Subject line
        text: "please save private key", // plaintext body
        html: "please save private key", // html body
        attachments: [
            {   // file on disk as an attachment
                filename: email+'.pem',
                path: "../"+ email + '.pem' // stream this file
            }
        ]
    };
    
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