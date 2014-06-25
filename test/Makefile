TESTS = server.js
REPORTER = dot

test:
	@./node_modules/.bin/mocha \
		--reporter $(REPORTER) \
		--timeout 10s \
		--bail \
		$(TESTS)

.PHONY: test
