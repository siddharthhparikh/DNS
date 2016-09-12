
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

	$(document).click(function (event) {
		if (!$(event.target).is('.menu-list') && !$(event.target).is('.hamburger')) {
			$(".menu-list").animate({ height: 'toggle', width: 'toggle' }, 'fast');
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

    if (window.location.pathname.toLowerCase() == "/buyasset") {
        $('#registerAssetForm input[name="asset"]').val("A" + Date.now());
    } else if (window.location.pathname.toLowerCase() == "/tradeasset") {
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

    $("#registerAssetForm").submit(function (event) {
        event.preventDefault();

        if ($('#registerAssetForm input[name="asset"]').val().length > 0 && $('#registerAssetForm input[name="assetDescription"]').val().length > 0 && $('#registerAssetForm input[name="assetLength"]').val().length > 0 && $('#registerAssetForm input[name="assetValue"]').val().length > 0 && $('#registerAssetForm input[name="assetAmount"]').val().length > 0) {
            $("#RegisterAssetResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#RegisterAssetResults').css("visibility", "visible");
            sendMessage(JSON.stringify({ fcn: "asset_init", type: "invoke", args: [sessionUser, $('#registerAssetForm input[name="asset"]').val(), $('#registerAssetForm input[name="assetDescription"]').val(), $('#registerAssetForm input[name="assetLength"]').val(), $('#registerAssetForm input[name="assetValue"]').val(), $('#registerAssetForm input[name="assetAmount"]').val()] }));
            $("#registerAssetForm .btn").attr("disabled", "");
        } else {
            $('#RegisterAssetResults').css("visibility", "visible");
            $('#RegisterAssetResults').text("Error: Some fields are blank");
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

    $("#lookupUserForm").submit(function (event) {
        event.preventDefault();

        if ($('#lookupUserForm input[name="username"]').val().length > 0) {
            $("#LookupUserResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#LookupUserResults').css("visibility", "visible");
            sendMessage(JSON.stringify({ fcn: "profile_record", type: "query", args: [$('#lookupUserForm input[name="username"]').val()] }));
            $("#lookupUserForm .btn").attr("disabled", "");
        } else {
            $('#LookupUserResults').css("visibility", "visible");
            $('#LookupUserResults').text("Error: Some fields are blank");
        }
    });

    $("#lookupAssetForm").submit(function (event) {
        event.preventDefault();

        if ($('#lookupAssetForm input[name="asset"]').val().length > 0) {
            $("#LookupAssetResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#LookupAssetResults').css("visibility", "visible");
            sendMessage(JSON.stringify({ fcn: "asset_record", type: "query", args: [$('#lookupAssetForm input[name="asset"]').val()] }));
        } else {
            $('#LookupAssetResults').css("visibility", "visible");
            $('#LookupAssetResults').text("Error: Some fields are blank");
        }

        $("#lookupAssetForm .btn").attr("disabled", "");
    });

    $("#lookupAssetUsersForm").submit(function (event) {
        event.preventDefault();

        if ($('#lookupAssetUsersForm input[name="username"]').val().length > 0) {
            $("#LookupUsersAssetResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#LookupUsersAssetResults').css("visibility", "visible");
            sendMessage(JSON.stringify({ fcn: "user_asset_records", type: "query", args: [$('#lookupAssetUsersForm input[name="username"]').val()] }));
        } else {
            $('#LookupUsersAssetResults').css("visibility", "visible");
            $('#LookupUsersAssetResults').text("Error: Some fields are blank");
        }

        $("#lookupAssetUsersForm .btn").attr("disabled", "");
    });

    $("#profileForm").submit(function (event) {
        event.preventDefault();
        $('#ProfileResults').css("visibility", "visible");
        $("#ProfileResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        sendMessage(JSON.stringify({
            fcn: "profile_update", type: "invoke", args: [sessionUser, JSON.stringify({
                firstName: $('#profileForm input[name="profile_firstname"]').val(),
                lastName: $('#profileForm input[name="profile_lastname"]').val(),
                phoneNumber: $('#profileForm input[name="profile_phonenumber"]').val(),
                streetAddress: $('#profileForm input[name="profile_streetaddress"]').val(),
                city: $('#profileForm input[name="profile_city"]').val(),
                state: $('#profileForm input[name="profile_state"]').val(),
                postalCode: $('#profileForm input[name="profile_postalcode"]').val(),
                country: $('#profileForm select[name="profile_country"]').val()
            })]
        }));
        $("#profileForm .btn").attr("disabled", "");
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

    ws = new WebSocket(wsUri);
    ws.onopen = function (evt) {
        waitTime = 1000;

        $(".resultBox").css("visibility", "hidden");
        sendMessage(JSON.stringify({ fcn: "check_chain", type: "other", args: [] }));

        if (window.location.pathname.toLowerCase() == "/") {
            sendMessage(JSON.stringify({ fcn: "query_stats", type: "query", args: [] }));
        } else if (window.location.pathname.toLowerCase() == "/myassets" || window.location.pathname.toLowerCase() == "/tradeasset") {
            sendMessage(JSON.stringify({ fcn: "user_asset_records", type: "query", args: [sessionUser] }));
        } else if (window.location.pathname.toLowerCase() == "/mytrades") {
            sendMessage(JSON.stringify({ fcn: "user_trade_records", type: "query", args: [sessionUser] }));
        } else if (window.location.pathname.toLowerCase() == "/myaccount") {
            sendMessage(JSON.stringify({ fcn: "profile_record", type: "query", args: [sessionUser] }));
        }
    };
    ws.onclose = function (evt) {
        setTimeout(connect, waitTime);
        $(".resultBox").css("visibility", "visible");
        $(".resultBox").html("Attempting to reconnect to server in " + (waitTime / 1000) + " seconds <div class=\"loading-dot-reconnect\"></div><div class=\"loading-dot-reconnect second\"></div><div class=\"loading-dot-reconnect third\"></div>");
        waitTime = waitTime + 1000;
    };
    ws.onmessage = function (evt) {
        console.log('received ws msg:', evt.data);
        if (isJsonString(evt.data)) {
            processMsg(evt.data);
        }
    };
    ws.onerror = function (evt) {
        console.log(evt);
        $(".resultBox").css("visibility", "visible");
        $(".resultBox").html("Please refresh the page.");
    };
}

function processMsg(msg) {
    var converted = JSON.parse(msg);
    console.log('processing ws msg:', converted);
    if (converted.fcn == "query_stats") {
        if (isJsonString(bin2String(converted.message.data))) {
            var messageConverted = JSON.parse(bin2String(converted.message.data));

            $("#registeredProfilesStats").text(messageConverted.registeredProfiles);
            $("#registeredAssetsStats").text(messageConverted.registeredAssets);
            $("#openAssetTradesStats").text(messageConverted.openAssetTrades);
            $("#closeAssetTradesStats").text(messageConverted.closeAssetTrades);
            $("#makeOfferAssetTradesStats").text(messageConverted.makeOfferAssetTrades);
            $("#acceptedAssetTradesStats").text(messageConverted.acceptAssetTrades);
            $("#totalRegisteredAssetsStats").text(messageConverted.totalRegisteredAssets);
            $("#totalRenewAssetsStats").text(messageConverted.totalRenewAssets);
            $("#totalOpenAssetTradesStats").text(messageConverted.totalOpenAssetTrades);
        }
    } else if (converted.fcn == "asset_record") {
        if (isJsonString(bin2String(converted.message.data))) {
            var messageConverted = JSON.parse(bin2String(converted.message.data));
            var assetResults = "";

            if (window.location.pathname.toLowerCase() == "/lookup") {
                $.each(messageConverted, function (index, value) {
                    assetResults = assetResults + "<span class=\"resultBoxTitle\">" + toTitleCase(index) + ":</span> " + value.toString().replace("\n", "") + "<br>";
                });

                $('#LookupAssetResults').html(assetResults);
                $("#lookupAssetForm .btn").removeAttr("disabled");
            } else if (window.location.pathname.toLowerCase() == "/mytrades") {
                assetStorage.push(messageConverted);
            }
        } else if (converted.message == "error") {
            $('#LookupAssetResults').html(converted.error.msg);
            $("#lookupAssetForm .btn").removeAttr("disabled");
        }
    } else if (converted.fcn == "request_private_key") {
        if (window.location.pathname.toLowerCase() == "/myaccount") {
            $('#ProfileKeysResults').html("<span class=\"resultBoxTitle\">Private Key:</span> " + converted.message.replace(/\s\s+/g, ''));
        }
    } else if (converted.fcn == "profile_record") {
        if (isJsonString(bin2String(converted.message.data))) {
            var messageConverted = JSON.parse(bin2String(converted.message.data));
            var profileResults = "";

            if (window.location.pathname.toLowerCase() == "/myaccount") {
                var profileJsonData;

                if (isJsonString(messageConverted.jsonData)) {
                    profileJsonData = JSON.parse(messageConverted.jsonData);
                }

                if (profileJsonData) {
                    $('#profileForm input[name="profile_firstname"]').val(profileJsonData.firstName);
                    $('#profileForm input[name="profile_lastname"]').val(profileJsonData.lastName);
                    $('#profileForm input[name="profile_phonenumber"]').val(profileJsonData.phoneNumber);
                    $('#profileForm input[name="profile_streetaddress"]').val(profileJsonData.streetAddress);
                    $('#profileForm input[name="profile_city"]').val(profileJsonData.city);
                    $('#profileForm input[name="profile_state"]').val(profileJsonData.state);
                    $('#profileForm input[name="profile_postalcode"]').val(profileJsonData.postalCode);
                    $('#profileForm select[name="profile_country"]').val(profileJsonData.country);
                }

                $('#ProfileKeysResults').html("<span class=\"resultBoxTitle\">Public Key:</span> " + messageConverted.publicKey.replace(/\s\s+/g, ''));
            } else if (window.location.pathname.toLowerCase() == "/lookup") {
                if (converted.message == "error") {
                    $('#LookupUserResults').html(converted.error.msg);
                    $("#lookupUserForm .btn").removeAttr("disabled");
                } else {
                    $.each(messageConverted, function (index, value) {
                        if (index == "jsonData") {
                            if (isJsonString(value)) {
                                var profileJsonData = JSON.parse(value);
                                $.each(profileJsonData, function (cindex, cvalue) {
                                    if (cvalue.length > 0) {
                                        profileResults = profileResults + "<span class=\"resultBoxTitle\">" + toTitleCase(cindex) + ":</span> " + cvalue + "<br>";
                                    }
                                });
                            }
                        } else if (index != "credits") {
                            profileResults = profileResults + "<span class=\"resultBoxTitle\">" + toTitleCase(index) + ":</span> " + value + "<br>";
                        }
                    });

                    $('#LookupUserResults').html(profileResults);
                    $("#lookupUserForm .btn").removeAttr("disabled");
                }
            }
        }
    } else if (converted.fcn == "user_asset_records") {
        if (window.location.pathname.toLowerCase() == "/tradeasset") {
            if (isJsonString(bin2String(converted.message.data))) {
                var messageConverted = JSON.parse(bin2String(converted.message.data));
                $('#openTradeForm select[name="assetTagList"]').empty();
                $('#openTradeForm select[name="assetTagList"]').append("<option value=\"\">Select an Asset</option>");
                $.each(messageConverted.assets, function (index, value) {
                    $('#openTradeForm select[name="assetTagList"]').append("<option value=\"" + value.name + "\">" + value.name + " | " + value.description + "</option>");
                });

                $('#openTradeForm select[name="assetTagList"]').val(currentAssetName);
            }
        } else if (window.location.pathname.toLowerCase() == "/lookup") {
            if (isJsonString(bin2String(converted.message.data))) {
                var messageConverted = JSON.parse(bin2String(converted.message.data));
                var assetResults = "";

                $.each(messageConverted.assets, function (index, value) {
                    $.each(value, function (cindex, cvalue) {
                        if (cindex == "value") {
                            assetResults = assetResults + "<span class=\"resultBoxTitle\">" + toTitleCase(cindex) + ":</span> $" + cvalue + "<br>";
                        } else if (cindex != "publicKey") {
                            assetResults = assetResults + "<span class=\"resultBoxTitle\">" + toTitleCase(cindex) + ":</span> " + cvalue + "<br>";
                        }
                    });

                    assetResults = assetResults + "<br>";
                    assetStorage.push(value);
                });

                $('#LookupUsersAssetResults').html(assetResults);
            } else if (converted.message == "error") {
                $('#LookupUsersAssetResults').html(converted.error.msg);
            }

            $("#lookupAssetUsersForm .btn").removeAttr("disabled");
        } else if (window.location.pathname.toLowerCase() == "/myassets") {
            if (isJsonString(bin2String(converted.message.data))) {
                var messageConverted = JSON.parse(bin2String(converted.message.data));
                var count = 0;
                var totalValue = 0.0;

                $("#myAssetTable tbody").empty();

                $.each(messageConverted.assets, function (index, value) {
                    var assetResults = "";

                    assetResults = assetResults + "<td>" + value.name + "</td>";
                    assetResults = assetResults + "<td>" + value.registered + "</td>";
                    assetResults = assetResults + "<td>" + value.expires + "</td>";
                    assetResults = assetResults + "<td>" + value.amount + "</td>";
                    assetResults = assetResults + "<td>$" + parseFloat(value.value).toFixed(2) + "</td>";

                    if (value.locked) {
                        assetResults = assetResults + "<td class=\"action-box\"><button id=\"renew-asset-" + value.name.trim() + "\" class=\"btn\" disabled>Renew</button> <button id=\"trade-asset-" + value.name.trim() + "\" class=\"btn\" disabled>Trade</button> <button id=\"delete-asset-" + value.name.trim() + "\" class=\"btn\" disabled>Sell</button></td>";
                    } else if (value.expired) {
                        assetResults = assetResults + "<td class=\"action-box\"><button id=\"renew-asset-" + value.name.trim() + "\" class=\"btn\" disabled>Renew</button> <button id=\"trade-asset-" + value.name.trim() + "\" class=\"btn\" disabled>Trade</button> <button id=\"delete-asset-" + value.name.trim() + "\" class=\"btn\">Sell</button></td>";
                    } else {
                        assetResults = assetResults + "<td class=\"action-box\"><button id=\"renew-asset-" + value.name.trim() + "\" class=\"btn\">Renew</button> <button id=\"trade-asset-" + value.name.trim() + "\" class=\"btn\">Trade</button> <button id=\"delete-asset-" + value.name.trim() + "\" class=\"btn\">Sell</button></td>";
                    }

                    if (assetLengthCost) {
                        totalValue += parseFloat(value.value) * parseFloat(value.amount) * parseFloat(value.timeSpan) * purchasedAssetRate;
                    } else {
                        totalValue += parseFloat(value.value) * parseFloat(value.amount);
                    }

                    $("#myAssetTable tbody").append("<tr> " + assetResults + " </tr>");
                    assetStorage.push(value);
                    count++;
                });

                $("#myAssetTableStats").html("<b>Total # of Assets:</b> " + count + " | <b>Total Value of Assets:</b> $" + totalValue.toFixed(2) + " | <b>Money:</b> $" + parseFloat(messageConverted.credits).toFixed(2));

                $("[id^=renew-asset-]").click(function () {
                    if (!tableLocked) {
                        if (confirm("Would you like to renew " + $(this).attr("id").replace("renew-asset-", "").trim() + "?") == true) {
                            var length = prompt("How many days would you like to renew " + $(this).attr("id").replace("renew-asset-", "").trim() + " for?", "1");
                            if (length != null && length != undefined && length.trim() != "") {
                                tableLocked = true;
                                $("#AssetTableResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
                                $('#AssetTableResults').css("visibility", "visible");
                                sendMessage(JSON.stringify({ fcn: "asset_renew", type: "invoke", args: [sessionUser, $(this).attr("id").replace("renew-asset-", "").trim(), length] }));
                            } else {
                                alert("Error: Invalid length");
                            }
                        }
                    } else {
                        alert("Please wait until the current operation is done.");
                    }
                });

                $("[id^=trade-asset-]").click(function () {
                    if (!tableLocked) {
                        if (confirm("Would you like to trade " + $(this).attr("id").replace("trade-asset-", "").trim() + "?") == true) {
                            window.location.href = encodeURI("/tradeasset?asset=" + $(this).attr("id").replace("trade-asset-", "").trim());
                        }
                    } else {
                        alert("Please wait until the current operation is done.");
                    }
                });

                $("[id^=delete-asset-]").click(function () {
                    if (!tableLocked) {
                        if (confirm("Would you like to sell " + $(this).attr("id").replace("delete-asset-", "").trim() + "?") == true) {
                            tableLocked = true;
                            $("#AssetTableResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
                            $('#AssetTableResults').css("visibility", "visible");
                            sendMessage(JSON.stringify({ fcn: "asset_delete", type: "invoke", args: [sessionUser, $(this).attr("id").replace("delete-asset-", "").trim()] }));
                        }
                    } else {
                        alert("Please wait until the current operation is done.");
                    }
                });

                if (count == 0) {
                    $("#myAssetTable tbody").append("<tr> <td colspan=\"6\">No assets have been purchased yet</td> </tr>");
                }
            } else if (converted.message == "error") {
                $("#myAssetTable tbody").append("<tr> <td colspan=\"6\">" + converted.error.msg + "</td> </tr>");
            }
        }
    } else if (converted.fcn == "user_trade_records") {
        if (isJsonString(bin2String(converted.message.data))) {
            var messageConverted = JSON.parse(bin2String(converted.message.data));
            var count = 0;
            var totalValue = 0.0;

            $("#myTradesTable tbody").empty();

            $.each(messageConverted.trades, function (index, value) {
                var tradeResults = "";

                tradeResults = tradeResults + "<td>" + value.assetName + "</td>";
                tradeResults = tradeResults + "<td>" + value.userIdB + "</td>";
                if (value.status == "waitforowner") {
                    tradeResults = tradeResults + "<td>Waiting for Owner</td>";
                } else {
                    tradeResults = tradeResults + "<td>Waiting for New Owner</td>";
                }
                tradeResults = tradeResults + "<td>" + value.amount + "</td>";
                tradeResults = tradeResults + "<td>$" + value.value + "</td>";

                if (value.currentOwner == sessionUser) {
                    if (value.status == "waitforowner") {
                        tradeResults = tradeResults + "<td class=\"action-box\"><button id=\"accept-trade-" + value.assetName.trim() + "\" class=\"btn\">Accept</button> <button id=\"decline-trade-" + value.assetName.trim() + "\" class=\"btn\">Decline</button> <button id=\"cancel-trade-" + value.assetName.trim() + "\" class=\"btn\">Cancel</button></td>";
                    } else {
                        tradeResults = tradeResults + "<td class=\"action-box\"><button id=\"accept-trade-" + value.assetName.trim() + "\" class=\"btn\" disabled>Accept</button> <button id=\"decline-trade-" + value.assetName.trim() + "\" class=\"btn\" disabled>Decline</button> <button id=\"cancel-trade-" + value.assetName.trim() + "\" class=\"btn\">Cancel</button></td>";
                    }
                } else {
                    if (value.status == "waitforowner") {
                        tradeResults = tradeResults + "<td class=\"action-box\"><button id=\"accept-trade-" + value.assetName.trim() + "\" class=\"btn\" disabled>Accept</button> <button id=\"decline-trade-" + value.assetName.trim() + "\" class=\"btn\" disabled>Decline</button> <button id=\"make-offer-trade-" + value.assetName.trim() + "\" class=\"btn\" disabled>Make a Offer</button></td>";
                    } else {
                        tradeResults = tradeResults + "<td class=\"action-box\"><button id=\"accept-trade-" + value.assetName.trim() + "\" class=\"btn\">Accept</button> <button id=\"decline-trade-" + value.assetName.trim() + "\" class=\"btn\">Decline</button> <button id=\"make-offer-trade-" + value.assetName.trim() + "\" class=\"btn\">Make a Offer</button></td>";
                    }
                }

                if (assetLengthCost) {
                    totalValue += parseFloat(value.value) * parseFloat(value.amount) * parseFloat(value.timeSpan) * purchasedAssetRate;
                } else {
                    totalValue += parseFloat(value.value) * parseFloat(value.amount);
                }

                sendMessage(JSON.stringify({ fcn: "asset_record", type: "query", args: [value.assetName] }));
                $("#myTradesTable tbody").append("<tr> " + tradeResults + " </tr>");
                count++;
            });

            $("#myTradesTableStats").html("<b>Total # of Trades:</b> " + count + " | <b>Total Value of Trades:</b> $" + totalValue.toFixed(2) + " | <b>Money:</b> $" + parseFloat(messageConverted.credits).toFixed(2));

            $("[id^=accept-trade-]").click(function () {
                var assetInfoData = findAssetInformation($(this).attr("id").replace("accept-trade-", "").trim());

                if (assetInfoData) {
                    $("#TradeTableResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
                    $('#TradeTableResults').css("visibility", "visible");
                    sendMessage(JSON.stringify({ fcn: "asset_transfer_accept", type: "invoke", args: [sessionUser, $(this).attr("id").replace("accept-trade-", "").trim(), assetInfoData.description, assetInfoData.timeSpan, assetInfoData.value, assetInfoData.amount] }));
                }
            });

            $("[id^=decline-trade-]").click(function () {
                $("#TradeTableResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
                $('#TradeTableResults').css("visibility", "visible");
                sendMessage(JSON.stringify({ fcn: "asset_transfer_decline", type: "invoke", args: [sessionUser, $(this).attr("id").replace("decline-trade-", "").trim()] }));
            });

            $("[id^=make-offer-trade-]").click(function () {
                var amount = prompt("Enter your offer: (Ex. 10.00)" + $(this).attr("id").replace("make-offer-trade-", "").trim() + "?", "");
                if (amount != null && amount != undefined && amount.trim() != "") {
                    $("#TradeTableResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
                    $('#TradeTableResults').css("visibility", "visible");
                    sendMessage(JSON.stringify({ fcn: "asset_transfer_make_offer", type: "invoke", args: [sessionUser, $(this).attr("id").replace("make-offer-trade-", "").trim(), amount.trim()] }));
                }
            });

            $("[id^=cancel-trade-]").click(function () {
                $("#TradeTableResults").html("Sent Request! Waiting for response <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
                $('#TradeTableResults').css("visibility", "visible");
                sendMessage(JSON.stringify({ fcn: "asset_transfer_cancel", type: "invoke", args: [sessionUser, $(this).attr("id").replace("cancel-trade-", "").trim()] }));
            });

            if (count == 0) {
                $("#myTradesTable tbody").append("<tr> <td colspan=\"6\">No trades have been open yet</td> </tr>");
            }
        } else if (converted.message == "error") {
            $("#myTradesTable tbody").append("<tr> <td colspan=\"6\">" + converted.error.msg + "</td> </tr>");
        }
    } else if (converted.fcn == "asset_init") {
        if (converted.message == "submitted") {
            $("#RegisterAssetResults").html("Processing asset request <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        } else if (converted.message == "complete") {
            $("#RegisterAssetResults").text("Your asset is now registered.");

            if (window.location.pathname.toLowerCase() == "/buyasset") {
                setTimeout(function () {
                    window.location.replace("/myassets");
                }, 500);
            }
        } else if (converted.message == "error") {
            $("#RegisterAssetResults").text("Oh no! An error has happen.");
            $("#registerAssetForm .btn").removeAttr("disabled");
        }
    } else if (converted.fcn == "asset_transfer_init") {
        if (converted.message == "submitted") {
            $("#OpenTradeResults").html("Processing trade request <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        } else if (converted.message == "complete") {
            $("#OpenTradeResults").text("Your trade is now open.");

            if (window.location.pathname.toLowerCase() == "/tradeasset") {
                setTimeout(function () {
                    window.location.replace("/mytrades");
                }, 500);
            }
        } else if (converted.message == "error") {
            $("#OpenTradeResults").text("Oh no! An error has happen.");
        }
    } else if (converted.fcn == "asset_renew" || converted.fcn == "asset_delete") {
        if (converted.message == "submitted") {
            $("#AssetTableResults").html("Processing asset request <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        } else if (converted.message == "complete") {
            $("#AssetTableResults").text("Completed your request.");
            tableLocked = false;
        } else if (converted.message == "error") {
            $("#AssetTableResults").text("Oh no! An error has happen.");
        }
    } else if (converted.fcn == "profile_init") {
        if (converted.message == "submitted") {
            $("#CreateAccountResults").html("Registering account with the chain <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
            $('#CreateAccountResults').css("visibility", "visible");
        } else if (converted.message == "register_completed") {
            $("#CreateAccountResults").removeClass("resultBox");
            $("#CreateAccountResults").text("Username: " + converted.username + " Password: " + converted.password);
            $('#CreateAccountResults').css("visibility", "visible");;
            $("#createAccountForm .btn").removeAttr("disabled");
        } else if (converted.message == "error") {
            $("#CreateAccountResults").text("Oh no! An error has happen.");
            $('#CreateAccountResults').css("visibility", "visible");;
            $("#createAccountForm .btn").removeAttr("disabled");
        } else if (converted.messageType == "waiting") {
            $("#CreateAccountResults").html(converted.message + "<div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        } else {
            $("#CreateAccountResults").text(converted.message);
            $("#createAccountForm .btn").removeAttr("disabled");
        }
    } else if (converted.fcn == "profile_update") {
        if (converted.message == "submitted") {
            $("#ProfileResults").html("Processing profile changes <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        } else if (converted.message == "complete") {
            $("#ProfileResults").html("Updated profile.");
            $("#profileForm .btn").removeAttr("disabled");
        } else if (converted.message == "error") {
            $("#ProfileResults").html("Oh no! An error has happen.");
            $("#profileForm .btn").removeAttr("disabled");
        }
    } else if (converted.fcn == "asset_transfer_accept" || converted.fcn == "asset_transfer_decline" || converted.fcn == "asset_transfer_cancel" || converted.fcn == "asset_transfer_make_offer") {
        if (converted.message == "submitted") {
            $("#TradeTableResults").html("Processing trade <div class=\"loading-dot\"></div><div class=\"loading-dot second\"></div><div class=\"loading-dot third\"></div>");
        } else if (converted.message == "complete") {
            $("#TradeTableResults").html("Trade processed.");
        } else if (converted.message == "error") {
            $("#TradeTableResults").html("Oh no! An error has happen.");
        }
    } else if (converted.fcn == "check_chain") {
        if (converted.message == "online") {
            $(".notification-container").remove();
        } else if ($(".notification-container").length == 0) {
            $("body").append("<div class=\"notification-container\"><div class=\"notification-box\">Waiting for the blockchain<br> <div class=\"loading-dot-blockchain\"></div><div class=\"loading-dot-blockchain second\"></div><div class=\"loading-dot-blockchain third\"></div></div></div>");
        }
    }
}

function bin2String(array) {
    return String.fromCharCode.apply(String, array);
}

function isJsonString(str) {
    try {
        JSON.parse(str);
    } catch (e) {
        return false;
    }

    return true;
}

function sendMessage(message) {
    console.log("SENT: " + message);
    ws.send(message);
}