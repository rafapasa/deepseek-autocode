# 1. Criar estrutura
chmod +x build.sh
./build.sh

# 2. Configurar API Key
export DEEPSEEK_API_KEY="<YOUR_API_KEY>"

# 3. Compilar
cd deepseek-autocode
go mod tidy
go build -o deepseek-autocode ./cmd/deepseek-autocode

# 4. Executar
./deepseek-autocode "Projeto Front-OpenERP Dart+Flutter, melhorar a interface e apresentação visual da tela que lista as " "lib/..."


