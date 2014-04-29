TESTS = server.js
# TEST = index.js
REPORTER = dot

test:
	@./node_modules/.bin/mocha \
		--reporter $(REPORTER) \
		--timeout 5000ms \
		--bail \
		$(TESTS)

.PHONY: test
