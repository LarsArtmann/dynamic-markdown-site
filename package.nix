{
  lib,
  buildGoModule,
  templ,
  go_1_27,
}:

let
  pname = "dynamic-markdown-site";
  version = "0.0.0";
in
(buildGoModule.override { go = go_1_27; }) {
  inherit pname version;

  vendorHash = import ./vendorHash.nix;

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./cmd
      ./internal
      ./go.mod
      ./go.sum
      ./templates
    ];
  };

  nativeBuildInputs = [ templ ];

  preBuild = ''
    templ generate
  '';

  env = {
    CGO_ENABLED = 0;
    GOEXPERIMENT = "jsonv2";
  };
  doCheck = false;
  tags = [
    "netgo"
    "osusergo"
  ];

  ldflags = [
    "-s"
    "-w"
  ];

  meta = with lib; {
    description = "Blazing-fast markdown site generator with live reload";
    homepage = "https://github.com/LarsArtmann/dynamic-markdown-site";
    license = licenses.unfree;
    mainProgram = pname;
  };
}
