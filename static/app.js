/**
 * Plaxt Frontend Application
 * Handles device authentication, modal dialogs, and wizard navigation
 */

// Device Authentication Polling
function pollDeviceAuth(deviceCode, interval) {
    $.ajax({
        url: "/api/auth/device/poll?device_code=" + deviceCode,
        success: function (data, textStatus, xhr) {
            if (xhr.status === 202) {
                // Pending, wait and retry
                setTimeout(function () {
                    pollDeviceAuth(deviceCode, interval);
                }, interval * 1000);
            } else {
                // Success! Reload to dashboard
                window.location.reload();
            }
        },
        error: function (xhr) {
            var msg = "Authentication failed";
            if (xhr.responseText) msg += ": " + xhr.responseText;
            $("#device-auth-error").text(msg).show();
        },
    });
}

var isAuthReady = false;

function prepareDeviceAuth() {
    $.getJSON("/api/auth/device/code", function (data) {
        isAuthReady = true;

        // Setup code display
        $("#device-code-display").text(data.user_code);

        // Construct complete URL
        var verifyUrl = data.verification_url_complete;
        if (!verifyUrl && data.verification_url) {
            verifyUrl = data.verification_url;
            if (verifyUrl.endsWith("/")) verifyUrl = verifyUrl.slice(0, -1);
            verifyUrl += "/" + data.user_code;
        }

        // Update Modal Link
        $("#device-verify-link").attr("href", verifyUrl);

        // Update Main Button
        var btn = $(".js-authorise");
        btn.attr("href", verifyUrl);
        btn.removeClass("disabled").text("Connect with Trakt");

        // Start Polling
        pollDeviceAuth(data.device_code, data.interval);
    }).fail(function () {
        $(".js-authorise").text("Error loading auth. Reload page.");
    });
}

// Clipboard Functions
function copyToClipboard(elementId, btnElement) {
    var copyText = document.getElementById(elementId);
    navigator.clipboard.writeText(copyText.innerText).then(
        function () {
            var originalText = btnElement.innerText;
            btnElement.innerText = "Copied!";
            setTimeout(function () {
                btnElement.innerText = originalText;
            }, 2000);
        },
        function (err) {
            console.error("Could not copy text: ", err);
        }
    );
}

function copyDeviceCode() {
    copyToClipboard(
        "device-code-display",
        document.getElementById("copy-code-btn")
    );
}

function copyWebhook() {
    copyToClipboard("webhook-url", document.querySelector(".webhook-box .copy-btn"));
}

function copyWebhookWizard() {
    copyToClipboard("webhook-url-wizard", document.querySelector("#step-2 .copy-btn"));
}

// jQuery Ready Handler
$(document).ready(function () {
    // Device Auth Setup
    $(".js-authorise").click(function (e) {
        if (!isAuthReady) {
            e.preventDefault();
            return;
        }
        // Show modal
        var modal = $("#device-auth-modal");
        modal.css("display", "flex").hide().fadeIn(300);
    });

    if ($(".js-authorise").length > 0) {
        prepareDeviceAuth();
    }

    // Device Auth Modal Cancel
    $("#device-auth-cancel").click(function () {
        $("#device-auth-modal").fadeOut(400);
    });

    // Logout Modal Logic
    var modal = $("#logout-modal");
    $(".js-logout-trigger").click(function () {
        modal.css("display", "flex").hide().fadeIn(300);
    });

    $(".js-modal-cancel").click(function () {
        modal.fadeOut(400);
    });

    $(window).click(function (e) {
        if ($(e.target).is(modal)) {
            modal.fadeOut(400);
        }
    });

    // Wizard Navigation
    console.log("Binding .js-next handler, found:", $(".js-next").length, "buttons");
    $(".js-next").click(function (e) {
        console.log("Next button clicked!");
        e.preventDefault();
        var current = $(this).closest(".wizard-step");
        var nextId = $(this).data("next");
        console.log("Next ID:", nextId, "Current:", current.attr("id"));

        current.hide();
        $("#" + nextId).fadeIn(300);

        // Update Progress
        $("#progress-webhook").addClass("completed").removeClass("active");
        $("#progress-webhook .step-icon").text("✓");
        $("#progress-config").addClass("active");
        $(".step-line").addClass("active");
    });

    $(".js-back").click(function (e) {
        e.preventDefault();
        var current = $(this).closest(".wizard-step");
        var prevId = $(this).data("prev");

        current.hide();
        $("#" + prevId).fadeIn(300);

        // Revert Progress
        $("#progress-webhook").removeClass("completed").addClass("active");
        $("#progress-webhook .step-icon").text("2");
        $("#progress-config").removeClass("active");
        $(".step-line").removeClass("active");
    });

    // Dashboard Webhook Toggle
    $(".js-toggle-webhook").click(function (e) {
        e.preventDefault();
        $(".webhook-drawer").slideToggle();
        $(this).text(function (i, text) {
            return text === "Show Webhook" ? "Hide Webhook" : "Show Webhook";
        });
    });

    // Reactive Save Preferences Button
    var $form = $("#preferences-form");
    var $submitBtn = $("#save-prefs-btn");

    if ($form.length && $submitBtn.length) {
        var initialState = $form.serialize();

        $form.on("change input", "input", function () {
            var currentState = $form.serialize();
            $submitBtn.prop("disabled", currentState === initialState);
        });
    }
});
