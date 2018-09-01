// INIT

// Load Foundation
$(document).foundation();
// Insure user action map hot areas are in the right place
imageMapResize();
// Continuously update elements for current
// date and time with current local values
setInterval(update_displayed_date, 1000);
function update_displayed_date(){
    let current_timestamp = new Date();
    // Plus one for the month because it's the only
    // zero based calendar attribute 🤦🏾‍♂:face-palm:️
    // https://stackoverflow.com/questions/1507619/javascript-date-utc-function-is-off-by-a-month
    let current_date = current_timestamp.getFullYear() + '/' +(current_timestamp.getMonth() + 1) + '/' + current_timestamp.getDate();
    let current_time = current_timestamp.getHours() + ':' + current_timestamp.getMinutes() + ':' + current_timestamp.getSeconds();
    $("#current-date > a").text(current_date);
    $("#current-time > a").text(current_time);
}
// For local development/avoiding infinite routing loops
if (window.location.search === "?local") {
    $("#what-am-I-doing")[0].href = "https://10.0.0.139:8080";
}

// REACT

// Set event handler for
// user connection actions
$(".do-connect-button").click(function(){
    do_connect();
});

// DATA

// eXecute In Place data for
// Extraction and reseting of
// connection info
let connection_extract_and_reset_xip = [
    {
        selector: '#connect-message',
        index: 0,
        accesor: 'value',
        reset_to: "''",
        store_as: 'message'
    },
    {
        selector: '#email-connect-id',
        index: 0,
        accesor: 'value',
        reset_to: "''",
        store_as: 'sender_email'
    },
    {
        selector: '#text-connect-id',
        index: 0,
        accesor: 'value',
        reset_to: "''",
        store_as: 'sender_phone'
    }
];

// LIBRARY

// Execute user connection requests
function do_connect() {
    // Extract and reset user connection data
    let connection_info = extract_and_reset_connection_request(connection_extract_and_reset_xip);
    let connection_type = "";
    // Validate either
    // no connect-id is provided
    // in which case default to sending an email
    // more than one connect-id is provided
    // in which case return an error to the user
    // or one connect-id is provided
    // in which case use it.
    if (connection_info.sender_phone === "" && connection_info.sender_email === "") {
        connection_type = "email";
        connection_info.sender = "";
    } else if (connection_info.sender_phone !== "" && connection_info.sender_email !== "") {
        alert("Please provide only one of email or phone number");
        return false
    } else if(connection_info.sender_phone !== "") {
        connection_type = "sms";
        connection_info.sender = connection_info.sender_phone;
    } else {
        connection_type = "email";
        connection_info.sender = connection_info.sender_email;
    }
    // Based off connect-id type, call appropriate
    // backend endpoint
    return call_backend("/"+connection_type, "POST", connection_info, "json",on_success, on_failure).then(function(response){
        console.log(`Successful connection: ${JSON.stringify(response)}`);
    });
    // Alert user to connection success
    // and reset input element to ready state
    function on_success(data){
        alert(`Connected with: ${JSON.stringify(connection_info)} Response: ${JSON.stringify(data)}`);
        setTimeout(function(){
        }, 2000);
    }
    // Alert user to connection failure
    function on_failure(data){
        alert(`Failed: ${JSON.stringify(data)}`);
    }
    return true
}
// Make an AJAX call to the desired backend endpoint
// with the specified data and event handlers
function call_backend(url, method, data, data_type, on_success, on_error){
    if(data_type === 'json'){
        data = JSON.stringify(data);
        content_type = "application/json; charset=utf-8";
    } else {
        content_type = "text/plain";
    }
    return $.ajax({
      method: method,
      url: url,
      data: data,
      contentType: content_type,
      dataType: data_type,
      success: on_success,
      error: on_error
    });
}
// Iterate and execute provided
// computational data structure
// (e.g. eXecute In Place)
// for connection fields to extract and reset
function extract_and_reset_connection_request(field_extract_reset_xips){
    let connection_request = {};
    for (let field_extract_reset_xip of field_extract_reset_xips){
        let extraction_target = field_extract_reset_xip.selector;
        let extraction_index = field_extract_reset_xip.index;
        let extraction_accessor = field_extract_reset_xip.accesor;
        let extracted_value = extract_element_value(extraction_target, extraction_index, extraction_accessor);
        let reset_value = field_extract_reset_xip.reset_to;
        reset_element_value(extraction_target,extraction_index,extraction_accessor,reset_value);
        connection_request[field_extract_reset_xip.store_as] =extracted_value;
    }
    return connection_request;
}
// Returns the input value for the specified
// HTML element
function extract_element_value(selector, index, accesor){
    // i.e. $('#email-connect')[0].value;
    return Function('return $("' + selector + '")[' + index +'].' + accesor)();
}
// Sets the display value for the specified
// HTML element
function reset_element_value(selector, index, accesor, reset){
    // i.e. $("#subscribe-to-mailing-list")[0].checked = false;
    return Function('return $("' + selector + '")[' + index +'].' + accesor + '=' + reset)();
}
