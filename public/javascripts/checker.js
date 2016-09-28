
/*******************************************************************************
 * Copyright (c) 2016 IBM Corp.
 *
 * All rights reserved. 
 *
 * Contributors:
 *   Justin E. Ervin - Initial implementation
 *******************************************************************************/

var sessionUser = "";
var currentAssetName = "";
var tableLocked = false;
var purchasedAssetRate = 0.00273972603;
var assetLengthCost = false;
var assetStorage = [];

function findAssetInformation(assetName) {
    for (var i = 0; i < assetStorage.length; i++) {
        if (assetStorage[i].name.toLowerCase() == assetName.trim().toLowerCase()) {
            return assetStorage[i];
        }
    }

    return null;
}

function setAuth(user, token) {
    sessionUser = user;

    if (user.trim().length == 0) {
        $('#whoAmI').css("visibility", "hidden");
    }
}

function setCurrentAssetName(name) {
    currentAssetName = name;
}

function toTitleCase(str) {
    return str.replace(/\w\S*/g, function (txt) { return txt.charAt(0).toUpperCase() + txt.substr(1); });
}

$(document).ready(function () {
    connect();




    $(".menu-list").hide();
    $(".hamburger").click(function () {
        $(".menu-list").animate({ height: 'toggle', width: 'toggle' }, 'fast');
    });

    $(".scroller").click(function () {
        var id = $(this).val();
        $('html, body').animate({
            scrollTop: $(id).offset().top
        }, 2000);
    });

    $(".scroller2").click(function () {
        var id = $(this).val();
        $('html, body').animate({
            scrollTop: $(id).offset().top
        }, 2000);
    });

    $(document).click(function (event) {
        if (!$(event.target).is('.menu-list') && !$(event.target).is('.hamburger') && !$(event.target).is('.menu-bar-item')) {
            $(".menu-list").animate({ height: 'hide', width: 'hide' }, 'fast');
        }
    });

    $("#show-user-publicKey").click(function () {
        sendMessage(JSON.stringify({ fcn: "profile_record", type: "query", args: [sessionUser] }));
        $('#ProfileKeysResults').css("visibility", "visible");
        $("#ProfileKeysResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
    });

    $("#show-user-privateKey").click(function () {
        var password = prompt("Enter your password: ", "");
        if (password != null && password != undefined && password.trim() != "") {
            sendMessage(JSON.stringify({ fcn: "request_private_key", type: "other", args: [sessionUser, password] }));
            $('#ProfileKeysResults').css("visibility", "visible");
            $("#ProfileKeysResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        }
    });

    if (window.location.pathname.toLowerCase() == "/buy") {
        $('#registerAssetForm input[name="asset"]').val("A" + Date.now());
    } else if (window.location.pathname.toLowerCase() == "/transfer") {
        $('#openTradeForm input[name="newAssetName"]').val("A" + Date.now())
    }

    $("#loginAccountForm").submit(function (event) {
        event.preventDefault();

        if ($('#loginAccountForm input[name="username"]').val().length > 0 && $('#loginAccountForm input[name="password"]').val().length > 0) {
            $("#LoginResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#LoginResults').css("visibility", "visible");
            $.post("/login", { username: $('#loginAccountForm input[name="username"]').val(), password: $('#loginAccountForm input[name="password"]').val() })
                .done(function (data) {
                    var converted = JSON.parse(data);

                    if (converted.message == "ok") {
                        window.location.href = '/';
                    } else {
                        $("#LoginResults").text("Error: Invalid username/password");
                        $("#loginAccountForm .btn").removeAttr("disabled");
                    }
                }).fail(function () {
                    $("#LoginResults").text("Error: Failed to complete request");
                    $("#loginAccountForm .btn").removeAttr("disabled");
                });

            $("#loginAccountForm .btn").attr("disabled", "");
        } else {
            $('#LoginResults').css("visibility", "visible");
            $('#LoginResults').text("Error: Some fields are blank");
        }
    });

    $("#createAccountForm").submit(function (event) {
        event.preventDefault();

        if ($('#createAccountForm input[name="username"]').val().length > 0) {
            $("#CreateAccountResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#CreateAccountResults').css("visibility", "visible");
            sendMessage(JSON.stringify({ fcn: "register", type: "other", args: [$('#createAccountForm input[name="username"]').val()] }));
            $("#createAccountForm .btn").attr("disabled", "");
        } else {
            $('#CreateAccountResults').css("visibility", "visible");
            $('#CreateAccountResults').text("Error: Some fields are blank");
        }
    });



    $("#openTradeForm").submit(function (event) {
        event.preventDefault();

        if ($('#openTradeForm select[name="assetTagList"]').val().length > 0 && $('#openTradeForm input[name="newOwnerUsername"]').val().length > 0 && $('#openTradeForm input[name="tradeValue"]').val().length > 0 && $('#openTradeForm input[name="tradeAmount"]').val().length > 0) {
            $("#OpenTradeResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#OpenTradeResults').css("visibility", "visible");
            sendMessage(JSON.stringify({ fcn: "asset_transfer_init", type: "invoke", args: [sessionUser, $('#openTradeForm select[name="assetTagList"]').val().trim(), $('#openTradeForm input[name="newOwnerUsername"]').val(), $('#openTradeForm input[name="tradeValue"]').val(), $('#openTradeForm input[name="tradeAmount"]').val(), $('#openTradeForm input[name="newAssetName"]').val()] }));
            $("#openTradeForm .btn").attr("disabled", "");
        } else {
            $('#OpenTradeResults').css("visibility", "visible");
            $('#OpenTradeResults').text("Error: Some fields are blank");
        }
    });







});

var ws = {};
var waitTime = 1000;

function connect() {
    var wsUri = '';

    $(".resultBox").html("Reconnecting <div class=\"loading-dot-reconnect\"></div><div class=\"loading-dot-reconnect second\"></div><div class=\"loading-dot-reconnect third\"></div>");
    console.log('protocol', window.location.protocol);
    if (window.location.protocol === 'https:') {
        wsUri = "wss://" + window.location.hostname + ":" + window.location.port;
    } else {
        wsUri = "ws://" + window.location.hostname + ":" + window.location.port;
    }


}


