import * as path from "path";
import * as os from "os";
import { ExtensionContext, workspace } from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

const GORELEASER_FILENAMES = [
  ".goreleaser.yml",
  ".goreleaser.yaml",
  "goreleaser.yml",
  "goreleaser.yaml",
];

export function activate(context: ExtensionContext): void {
  const serverPath = getServerPath(context);

  const serverOptions: ServerOptions = {
    run: { command: serverPath },
    debug: { command: serverPath },
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      {
        scheme: "file",
        language: "yaml",
        pattern: "**/.goreleaser.{yml,yaml}",
      },
      {
        scheme: "file",
        language: "yaml",
        pattern: "**/goreleaser.{yml,yaml}",
      },
    ],
    synchronize: {
      fileEvents: workspace.createFileSystemWatcher(
        "**/{.goreleaser,.goreleaser,goreleaser}.{yml,yaml}"
      ),
    },
  };

  client = new LanguageClient(
    "goreleaser-ls",
    "GoReleaser Language Server",
    serverOptions,
    clientOptions
  );

  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

function getServerPath(context: ExtensionContext): string {
  const config = workspace.getConfiguration("goreleaser-ls");
  const customPath = config.get<string>("serverPath");
  if (customPath) {
    return customPath;
  }

  const platform = os.platform();

  let binaryName = "goreleaser-ls";
  if (platform === "win32") {
    binaryName = "goreleaser-ls.exe";
  }

  return path.join(context.extensionPath, "bin", binaryName);
}
