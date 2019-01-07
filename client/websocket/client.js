(function(win) {
    const rawHeaderLen = 16;
    const packetOffset = 0;
    const headerOffset = 4;
    const verOffset = 6;
    const opOffset = 8;
    const seqOffset = 12;

    var Client = function(options) {
        var MAX_CONNECT_TIMES = 10;
        var DELAY = 15000;
        this.options = options || {};
        this.createConnect(MAX_CONNECT_TIMES, DELAY);
    }


    var appendMsg = function(text) {
        quoteRows = []
        quote = JSON.parse(text);
        if (quote.Code == "success") {
            var b = $('#box').DataTable({
                paging: false,
                ordering: false,
                info: false,
                searching: false,
                destroy:true
            });
            Result = quote.Result;
            // console.log(quote)
            symbol = Result[0];
            ask = Result[1];
            bid = Result[2];
            time = formatUnixtimestamp(Result[4] - 60*60*8);
            cAsk = $('#'+symbol).children('td').eq(2).text();
            cBid = $('#'+symbol).children('td').eq(1).text();


            dc = ask - bid;
            cdc = cAsk - cBid;
            // console.log(cAsk,ask)
            if (dc > cdc) {
                $('#'+symbol).children("td:gt(0)").css("color","red");
            } else {
                $('#'+symbol).children("td:gt(0)").css("color","blue");
            }
            // if (ask > cAsk) {
            //     $('#'+symbol).children('td').eq(1).css("color","red")
            // } else {
            //     $('#'+symbol).children('td').eq(1).css("color","green")
            // }
            // if (bid > cBid) {
            //     $('#'+symbol).children('td').eq(2).css("color","red")
            // } else {
            //     $('#'+symbol).children('td').eq(2).css("color","green")
            // }
            b.row($('#'+symbol)).data([symbol, bid, ask, time]);
        }
    }

    function formatUnixtimestamp (unixtimestamp){
        var unixtimestamp = new Date(unixtimestamp*1000);
        var year = 1900 + unixtimestamp.getYear();
        var month = "0" + (unixtimestamp.getMonth() + 1);
        var date = "0" + unixtimestamp.getDate();
        var hour = "0" + unixtimestamp.getHours();
        var minute = "0" + unixtimestamp.getMinutes();
        var second = "0" + unixtimestamp.getSeconds();
        return year + "-" + month.substring(month.length-2, month.length)  + "-" + date.substring(date.length-2, date.length)
            + " " + hour.substring(hour.length-2, hour.length) + ":"
            + minute.substring(minute.length-2, minute.length) + ":"
            + second.substring(second.length-2, second.length);
    }

    Client.prototype.createConnect = function(max, delay) {
        var self = this;
        if (max === 0) {
            return;
        }
        connect();

        var textDecoder = new TextDecoder();
        var textEncoder = new TextEncoder();
        var heartbeatInterval;
        function connect() {
            var ws = new WebSocket('ws://127.0.0.1:3102/sub');
            ws.binaryType = 'arraybuffer';
            ws.onopen = function() {
                auth();
            }

            ws.onmessage = function(evt) {
                var data = evt.data;
                var dataView = new DataView(data, 0);
                var packetLen = dataView.getInt32(packetOffset);
                var headerLen = dataView.getInt16(headerOffset);
                var ver = dataView.getInt16(verOffset);
                var op = dataView.getInt32(opOffset);
                var seq = dataView.getInt32(seqOffset);

                // console.log("receiveHeader: packetLen=" + packetLen, "headerLen=" + headerLen, "ver=" + ver, "op=" + op, "seq=" + seq);

                switch(op) {
                    case 8:
                        // auth reply ok
                        document.getElementById("status").innerHTML = "<color style='color:green'>ok<color>";
                        // send a heartbeat to server
                        heartbeat();
                        heartbeatInterval = setInterval(heartbeat, 30 * 1000);
                    break;
                    case 3:
                        // receive a heartbeat from server
                        // console.log("receive: heartbeat");
                        // appendMsg("receive: heartbeat from server");
                    break;
                    default:
                        // batch message
                        for (var offset=0; offset<data.byteLength; offset+=packetLen) {
                            // parse
                            var packetLen = dataView.getInt32(offset);
                            var headerLen = dataView.getInt16(offset+headerOffset);
                            var ver = dataView.getInt16(offset+verOffset);
                            var msgBody = textDecoder.decode(data.slice(offset+headerLen, offset+packetLen));
                            // callback
                            messageReceived(ver, msgBody);
                            // appendMsg("receive: ver=" + ver + " op=" + op + " seq=" + seq + " message=" + msgBody);
                            appendMsg(msgBody);
                        }
                    break;
                }
            }

            ws.onclose = function() {
                if (heartbeatInterval) clearInterval(heartbeatInterval);
                setTimeout(reConnect, delay);

                document.getElementById("status").innerHTML =  "<color style='color:red'>failed<color>";
            }

            function heartbeat() {
                var headerBuf = new ArrayBuffer(rawHeaderLen);
                var headerView = new DataView(headerBuf, 0);
                headerView.setInt32(packetOffset, rawHeaderLen);
                headerView.setInt16(headerOffset, rawHeaderLen);
                headerView.setInt16(verOffset, 1);
                headerView.setInt32(opOffset, 2);
                headerView.setInt32(seqOffset, 1);
                ws.send(headerBuf);
                // console.log("send: heartbeat",headerBuf);
                // appendMsg("send: heartbeat to server");
            }

            function auth() {
                var token = '{"mid":123, "room_id":"live://1000", "platform":"web", "accepts":[1000,1001,1002]}'
                var headerBuf = new ArrayBuffer(rawHeaderLen);
                var headerView = new DataView(headerBuf, 0);
                var bodyBuf = textEncoder.encode(token);
                headerView.setInt32(packetOffset, rawHeaderLen + bodyBuf.byteLength);
                headerView.setInt16(headerOffset, rawHeaderLen);
                headerView.setInt16(verOffset, 1);
                headerView.setInt32(opOffset, 7);
                headerView.setInt32(seqOffset, 1);
                ws.send(mergeArrayBuffer(headerBuf, bodyBuf));

                document.getElementById("session").innerHTML = "auth: " + token;
            }

            function messageReceived(ver, body) {
                var notify = self.options.notify;
                if(notify) notify(body);
                // console.log("messageReceived:", "ver=" + ver, "body=" + body);
            }

            function mergeArrayBuffer(ab1, ab2) {
                var u81 = new Uint8Array(ab1),
                    u82 = new Uint8Array(ab2),
                    res = new Uint8Array(ab1.byteLength + ab2.byteLength);
                res.set(u81, 0);
                res.set(u82, ab1.byteLength);
                return res.buffer;
            }

            function char2ab(str) {
                var buf = new ArrayBuffer(str.length);
                var bufView = new Uint8Array(buf);
                for (var i=0; i<str.length; i++) {
                    bufView[i] = str[i];
                }
                return buf;
            }

        }

        function reConnect() {
            self.createConnect(--max, delay * 2);
        }
    }

    win['MyClient'] = Client;
})(window);
