#!/bin/bash
DEBUG_FILE="$1"
# Verifica se o arquivo existe
if [ ! -f "$DEBUG_FILE" ]; then
    echo "Erro: Arquivo $DEBUG_FILE não encontrado!"
    exit 1
fi

# Extrai o comando completo e remove quebras de linha, garantindo apenas um espaço
TF_REATTACH_FULL=$(grep -o "TF_REATTACH_PROVIDERS='.*'" "$DEBUG_FILE" | tr -d '\n' | tr -d '\r' | tr -s ' ')

# Detecta o sistema operacional e usa o comando apropriado
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    echo "$TF_REATTACH_FULL" | pbcopy
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    if command -v xclip &> /dev/null; then
        echo "export $TF_REATTACH_FULL" | xclip -selection clipboard
    elif command -v xsel &> /dev/null; then
        echo "export $TF_REATTACH_FULL" | xsel --clipboard
    else
        echo "Não foi possível encontrar xclip ou xsel. Instale um deles para copiar para o clipboard."
        echo "Valor completo: $TF_REATTACH_FULL"
        exit 1
    fi
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" || "$OSTYPE" == "win32" ]]; then
    # Windows
    echo "$TF_REATTACH_FULL" | clip
else
    echo "Sistema operacional não suportado para cópia automática."
    echo "Valor completo: $TF_REATTACH_FULL"
    exit 1
fi

echo "Comando 'TF_REATTACH_PROVIDERS=...' copiado para o clipboard!"