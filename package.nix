{
  lib,
  buildGoModule,
  templ,
}:

let
  pname = "dynamic-markdown-site";
  version = "0.0.0";
in
buildGoModule {
  inherit pname version;

  vendorHash = "sha256-/bIf2sea5gjbB8GFtl27yePL/BVP4paPr5eeKA4BLVo=";

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

  env.CGO_ENABLED = 0;
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
