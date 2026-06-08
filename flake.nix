{
  description = "Blazing-fast markdown site generator with live reload";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          system,
          ...
        }:
        let
          pkgs = import inputs.nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };
          pname = "dynamic-markdown-site";
          version = self.rev or self.dirtyRev or "dev";
          vendorHash = "sha256-/bIf2sea5gjbB8GFtl27yePL/BVP4paPr5eeKA4BLVo=";

          sourceFiles = lib.fileset.unions [
            ./cmd
            ./internal
            ./go.mod
            ./go.sum
            ./templates
          ];

          ldflags = [
            "-s"
            "-w"
            "-X github.com/larsartmann/dynamic-markdown-site/internal/version.Version=${version}"
            "-X github.com/larsartmann/dynamic-markdown-site/internal/version.Commit=${version}"
            "-X github.com/larsartmann/dynamic-markdown-site/internal/version.BuildDate=unknown"
          ];

          pkg = pkgs.buildGoModule {
            inherit
              pname
              version
              vendorHash
              ldflags
              ;

            src = lib.fileset.toSource {
              root = ./.;
              fileset = sourceFiles;
            };

            nativeBuildInputs = with pkgs; [ templ ];

            preBuild = ''
              templ generate
            '';

            env.CGO_ENABLED = 0;
            tags = [
              "netgo"
              "osusergo"
            ];

            doCheck = false;

            meta = with lib; {
              description = "Blazing-fast markdown site generator with live reload";
              homepage = "https://github.com/LarsArtmann/dynamic-markdown-site";
              license = licenses.unfree;
              mainProgram = pname;
              maintainers = [ maintainers.larsartmann ];
            };
          };
        in
        {
          _module.args.pkgs = pkgs;

          packages.default = pkg;

          checks = {
            format = config.treefmt.build.check self;
            build = config.packages.default;
            test = config.packages.default.overrideAttrs (_: { doCheck = true; });
          };

          devShells.default = pkgs.mkShell {
            inputsFrom = [ config.packages.default ];

            packages = builtins.attrValues {
              inherit (pkgs)
                golangci-lint
                gopls
                gotools
                templ
                ;
            };

            GOWORK = "off";
          };

          treefmt.settings = {
            formatter.nixfmt = {
              command = pkgs.nixfmt;
              includes = [ "*.nix" ];
            };
          };

        };

      flake = {
        overlays.default = final: _prev: {
          dynamic-markdown-site = final.callPackage ./package.nix { };
        };
      };
    };
}
