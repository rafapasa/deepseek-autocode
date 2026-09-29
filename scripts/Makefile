TAR  := /home/opc/sdk/go1.27.1/go1.27.1.linux-arm64.tar.gz
SUM  := 3450b45a3f9ee8568792736a5c5e70a1f2e9b36c35a8f74958c03e51d7d92bec

.PHONY: install-go
install-go:
	@echo "==> verificando tarball"
	@test -f "$(TAR)"
	@echo "$(SUM)  $(TAR)" | sha256sum -c -
	@echo "==> removendo pacotes Red Hat / dnf do Go"
	@-sudo dnf remove -y golang golang-bin golang-src golang-docs golang-race 2>/dev/null || true
	@-sudo rpm -e --nodeps golang golang-bin golang-src 2>/dev/null || true
	@echo "==> limpando /usr/local/go e bins antigos"
	@sudo rm -rf /usr/local/go
	@sudo rm -f /usr/bin/go /usr/bin/gofmt /usr/local/bin/go /usr/local/bin/gofmt
	@echo "==> instalando go1.27.1 em /usr/local/go"
	@sudo mkdir -p /usr/local
	@sudo tar -C /usr/local -xzf "$(TAR)"
	@sudo ln -sfn /usr/local/go/bin/go /usr/local/bin/go
	@sudo ln -sfn /usr/local/go/bin/gofmt /usr/local/bin/gofmt
	@printf '%s\n' 'export GOROOT=/usr/local/go' 'export PATH=/usr/local/go/bin:/usr/local/bin:$$PATH' | sudo tee /etc/profile.d/go.sh >/dev/null
	@sudo chmod 644 /etc/profile.d/go.sh
	@echo "==> versao"
	@/usr/local/go/bin/go version

.PHONY: check-go
check-go:
	@echo PATH=$$PATH
	@which go || true
	@go version || /usr/local/go/bin/go version
	@ls -l /usr/local/bin/go /usr/bin/go /usr/local/go/bin/go 2>/dev/null || true
	@rpm -q golang golang-bin 2>/dev/null || echo "rpm golang: ausente (ok)"

