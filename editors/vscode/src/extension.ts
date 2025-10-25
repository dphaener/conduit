import * as path from 'path';
import * as vscode from 'vscode';
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from 'vscode-languageclient/node';

let client: LanguageClient | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const config = vscode.workspace.getConfiguration('conduit');
  const enabled = config.get<boolean>('lsp.enabled', true);

  if (!enabled) {
    return;
  }

  // Get the conduit executable path from configuration
  const conduitPath = config.get<string>('lsp.path', 'conduit');

  // Server options: start the LSP server using conduit lsp command
  const serverOptions: ServerOptions = {
    command: conduitPath,
    args: ['lsp'],
    transport: TransportKind.stdio,
  };

  // Client options
  const clientOptions: LanguageClientOptions = {
    // Register the server for Conduit documents
    documentSelector: [
      {
        scheme: 'file',
        language: 'conduit',
      },
    ],
    synchronize: {
      // Notify the server about file changes to '.cdt' files
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.cdt'),
    },
  };

  // Create the language client
  client = new LanguageClient(
    'conduit-lsp',
    'Conduit Language Server',
    serverOptions,
    clientOptions
  );

  // Start the client (which also starts the server)
  try {
    await client.start();
    vscode.window.showInformationMessage('Conduit Language Server started');
  } catch (error) {
    vscode.window.showErrorMessage(
      `Failed to start Conduit Language Server: ${error}`
    );
  }
}

export async function deactivate(): Promise<void> {
  if (!client) {
    return;
  }

  await client.stop();
}
