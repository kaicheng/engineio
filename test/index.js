var listen = require('./common').listen

describe('base', function() {
	it('should start go server', function(done) {
		listen("default", function(port){done()})
	})
})