#standardSQL
CREATE TEMPORARY FUNCTION GetQueueStats( lines_json STRING)
RETURNS STRUCT<email STRING, executed1Minute FLOAT64>
LANGUAGE js AS """
var lines = JSON.parse(lines_json);
var result = {
    email : '',
    executed1Minute : 0
};
lines.forEach(function(element, index, array) {
    var emailPrefix = 'userEmail=';
    var queueStats = 'TaskQueue.QueueStats:';
    if (element.logMessage.startsWith(emailPrefix)) {
        result.email = element.logMessage.substring(emailPrefix.length);
    } else if (element.logMessage.startsWith(queueStats)) {
        var vj = element.logMessage.substring(queueStats.length);
        var v = JSON.parse(vj);
        result.executed1Minute = v.Executed1Minute;
    }
});

return result;
""";
SELECT
    insertId,
    timestamp,
    GetQueueStats(TO_JSON_STRING(protoPayload.line)) AS queueStats
FROM `appengine.appengine_googleapis_com_request_log_20171014`
ORDER BY
timestamp DESC
LIMIT 1000