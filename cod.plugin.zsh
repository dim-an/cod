local FOUND_RBENV=$+commands[cod]

if [[ $FOUND_RBENV -eq 1 ]]; then
  source <(cod init $$ zsh)
fi