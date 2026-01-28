{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:

{
  languages.go.enable = true;

  packages = with pkgs; [
    git
    golangci-lint
  ];

  git-hooks.hooks.golangci-lint.enable = true;
}
