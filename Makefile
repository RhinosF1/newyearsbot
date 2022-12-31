GOPATH=/usr/local/go/bin/go


install:
	install -Dm755 newyearsbot .
	install -Dm644 LICENSE "/usr/local/share/licenses/newyearsbot/LICENSE"

uninstall:
	rm "/usr/local/bin/newyearsbot"
	rm "/usr/local/share/licenses/newyearsbot/LICENSE"

clean:
	chmod -R 755 $(GOPATH)
	rm -rf $(GOPATH)
	rm newyearsbot
