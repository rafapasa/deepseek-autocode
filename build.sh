cd /home/opc/prj/deepseek-autocode
go build -o bin/deepseek-autocode ./cmd/deepseek-autocode
sudo restorecon -v bin/deepseek-autocode
sudo systemctl restart dsac
