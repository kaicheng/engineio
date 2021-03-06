
/**
 * Module dependencies.
 */

 var sys = require('sys')
 var exec = require('child_process').exec;
 var spawn = require('child_process').spawn;
 var util = require('util')
 var child

/**
 * Listen shortcut that fires a callback on an epheemal port.
 */

function getPort() {
	return Math.floor(Math.random() * (65000 - 1000) + 1000)
}

exports.listen = function (it, opts, fn) {
  if ('function' == typeof it) {
    fn = it;
    it = "default"
    opts = {}
  } else if ('string' != typeof it) {
    fn = opts
    opts = it
    it = "default"
  }
  if ('function' == typeof opts) {
    fn = opts;
    opts = {};
  }

  if (opts == null) {
  	opts = {};
  }

  var port = getPort()

  child = spawn(process.env["GOPATH"] + "/bin/test", [port, it, JSON.stringify(opts)], {stdio:[process.stdin, process.stdout, 'pipe']})

  child.stderr.on('data', function (data) {
    s = data.toString()
    if (s.indexOf("server ready") > -1) {
      fn(port)
      s = s.replace("server ready", "").trim()
    }
    process.stderr.write(s)
  })
  return null;
};

/**
 * Sprintf util.
 */

require('s').extend();
