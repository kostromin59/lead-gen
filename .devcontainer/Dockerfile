# Note: You can use any Debian/Ubuntu based image you want. 
FROM tdlib-go:latest

RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.zhs_history" \
  && echo "$SNIPPET" >> "/root/.zhsrc"

# [Optional] Uncomment this section to install additional OS packages.
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
  && apt-get -y install --no-install-recommends vim