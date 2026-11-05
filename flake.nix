{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      git-hooks,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        pre-commit = git-hooks.lib.${system}.run {
          src = ./.;
          hooks.golangci-lint.enable = true;
        };
      in
      {
        devShells.default = pkgs.mkShell {
          packages =
            with pkgs;
            [
              git
              go
              gopls
              golangci-lint
              ko
            ]
            ++ pre-commit.enabledPackages;

          env.GOTOOLCHAIN = "local";

          shellHook = pre-commit.shellHook;
        };
      }
    );
}
