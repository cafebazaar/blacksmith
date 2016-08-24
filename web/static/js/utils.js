var secondsToStr = function(seconds) {
    function numberEnding (number) {
        return (number > 1) ? 's' : '';
    }

    var temp = seconds;
    var res = "";
    var years = Math.floor(temp / 31536000);
    if (years) {
        res += years + ' year' + numberEnding(years) + ' ';
    }
    //TODO: Months! Maybe weeks? 
    var days = Math.floor((temp %= 31536000) / 86400);
    if (days) {
        res += days + ' day' + numberEnding(days) + ' ';
    }
    var hours = Math.floor((temp %= 86400) / 3600);
    if (hours) {
        res += hours + ' hour' + numberEnding(hours);
        if (years) return res;
        res += ' ';
    }
    var minutes = Math.floor((temp %= 3600) / 60);
    if (minutes) {
        res += minutes + ' minute' + numberEnding(minutes) + ' ';
        if (days) return res;
        res += ' ';
    }
    var seconds = Math.floor(temp % 60);
    if (seconds) {
        return res + seconds + ' second' + numberEnding(seconds);
    }
    if (res.length > 0) return res;
    return 'less than a second'; //'just now' //or other string you like;
};
