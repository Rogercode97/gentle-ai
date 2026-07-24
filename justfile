# Justfile para Gentle-AI (Termux Edition)

set shell := ["bash", "-c"]

# Muestra los comandos disponibles por defecto
default:
    @just --list

# Sincroniza tu fork con las actualizaciones oficiales de Gentleman-Programming
sync:
    @echo "🔄 Sincronizando con upstream (Gentleman-Programming)..."
    git fetch upstream
    @echo "🔀 Fusionando actualizaciones oficiales en tu rama actual..."
    git merge upstream/main
    @echo "☁️ Respaldando en tu fork (Rogercode97)..."
    git push rogercode HEAD:termux-main
    @echo "✅ Sincronización completada."

# Compila e instala localmente el binario en tu entorno Termux
build:
    @echo "🔨 Compilando gentle-ai..."
    go build -o gentle-ai ./cmd/gentle-ai
    @echo "📦 Instalando en ~/.local/bin..."
    install -m 755 gentle-ai ~/.local/bin/gentle-ai
    @echo "✅ Instalación completada."

# Comando todo-en-uno: sincroniza y luego compila e instala
update: sync build
    @echo "🚀 Actualización y despliegue local exitosos."
