
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

  child = spawn("go", ["run", "tester.go", port, it, JSON.stringify(opts)], {stdio:'inherit'})

  setTimeout(function() {fn(port)}, 2000)

  return null;
};

/**
 * Sprintf util.
 */

require('s').extend();
