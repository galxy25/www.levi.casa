// Init
// Load Foundation
$(document).foundation();
// Insure user action map hot areas are in the right place
imageMapResize();
// Continuously update placeholder elements for
// date and time with current local values
setInterval(update_displayed_date, 2);
function update_displayed_date(){
    let current_timestamp = new Date()
    // Plus one for the month because it's the only zero
    // based calendar attribute :facepalm:
    // https://stackoverflow.com/questions/1507619/javascript-date-utc-function-is-off-by-a-month
    let current_date = current_timestamp.getFullYear() + '/' +(current_timestamp.getMonth() + 1) + '/' + current_timestamp.getDate()
    let current_time = current_timestamp.getHours() + ':' + current_timestamp.getMinutes() + ':' + current_timestamp.getSeconds()
    $("#current-date > a").text(current_date)
    $("#current-time > a").text(current_time)
}
// React
