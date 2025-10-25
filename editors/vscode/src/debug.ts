import * as vscode from 'vscode';
import * as Net from 'net';
import { DebugProtocol } from '@vscode/debugprotocol';

/**
 * Debug adapter descriptor factory for Conduit debugger
 */
export class ConduitDebugAdapterDescriptorFactory implements vscode.DebugAdapterDescriptorFactory {
  createDebugAdapterDescriptor(
    session: vscode.DebugSession,
    executable: vscode.DebugAdapterExecutable | undefined
  ): vscode.ProviderResult<vscode.DebugAdapterDescriptor> {
    // Get configuration
    const config = vscode.workspace.getConfiguration('conduit');
    const conduitPath = config.get<string>('debug.path', 'conduit');
    const port = config.get<number>('debug.port', 0);

    if (port > 0) {
      // Connect to existing DAP server
      return new vscode.DebugAdapterServer(port);
    } else {
      // Launch new DAP server process
      return new vscode.DebugAdapterExecutable(conduitPath, ['debug', '--dap']);
    }
  }
}

/**
 * Configuration provider for Conduit debug sessions
 */
export class ConduitDebugConfigurationProvider implements vscode.DebugConfigurationProvider {
  /**
   * Resolve debug configuration before starting a debug session
   */
  resolveDebugConfiguration(
    folder: vscode.WorkspaceFolder | undefined,
    config: vscode.DebugConfiguration,
    token?: vscode.CancellationToken
  ): vscode.ProviderResult<vscode.DebugConfiguration> {
    // If launch.json is missing or empty
    if (!config.type && !config.request && !config.name) {
      const editor = vscode.window.activeTextEditor;
      if (editor && editor.document.languageId === 'conduit') {
        config.type = 'conduit';
        config.name = 'Launch Conduit Application';
        config.request = 'launch';
        config.program = '${workspaceFolder}/build/app';
        config.stopOnEntry = true;
      }
    }

    // Ensure required fields are present
    if (!config.program) {
      return vscode.window.showInformationMessage(
        'Cannot find a program to debug'
      ).then(_ => {
        return undefined; // abort launch
      });
    }

    // Resolve workspace folder variables
    if (folder && config.program) {
      config.program = config.program.replace('${workspaceFolder}', folder.uri.fsPath);
    }

    return config;
  }
}

/**
 * Initial debug configurations provider
 */
export class ConduitDebugConfigurationInitialProvider implements vscode.DebugConfigurationProvider {
  provideDebugConfigurations(
    folder: vscode.WorkspaceFolder | undefined,
    token?: vscode.CancellationToken
  ): vscode.ProviderResult<vscode.DebugConfiguration[]> {
    return [
      {
        type: 'conduit',
        request: 'launch',
        name: 'Launch Conduit Application',
        program: '${workspaceFolder}/build/app',
        stopOnEntry: false,
        args: [],
        cwd: '${workspaceFolder}',
        env: {},
        showLog: false,
      },
      {
        type: 'conduit',
        request: 'launch',
        name: 'Debug Conduit Application',
        program: '${workspaceFolder}/build/app',
        stopOnEntry: true,
        args: [],
        cwd: '${workspaceFolder}',
        env: {},
        showLog: true,
      },
    ];
  }
}

/**
 * Register debug support for Conduit
 */
export function activateDebug(context: vscode.ExtensionContext): void {
  // Register debug adapter descriptor factory
  context.subscriptions.push(
    vscode.debug.registerDebugAdapterDescriptorFactory(
      'conduit',
      new ConduitDebugAdapterDescriptorFactory()
    )
  );

  // Register debug configuration provider
  context.subscriptions.push(
    vscode.debug.registerDebugConfigurationProvider(
      'conduit',
      new ConduitDebugConfigurationProvider()
    )
  );

  // Register initial debug configurations provider
  context.subscriptions.push(
    vscode.debug.registerDebugConfigurationProvider(
      'conduit',
      new ConduitDebugConfigurationInitialProvider(),
      vscode.DebugConfigurationProviderTriggerKind.Initial
    )
  );

  // Register debug commands
  context.subscriptions.push(
    vscode.commands.registerCommand('conduit.debug.run', () => {
      vscode.debug.startDebugging(undefined, {
        type: 'conduit',
        name: 'Run Conduit Application',
        request: 'launch',
        program: '${workspaceFolder}/build/app',
        stopOnEntry: false,
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('conduit.debug.start', () => {
      vscode.debug.startDebugging(undefined, {
        type: 'conduit',
        name: 'Debug Conduit Application',
        request: 'launch',
        program: '${workspaceFolder}/build/app',
        stopOnEntry: true,
      });
    })
  );

  // Listen for debug session events
  context.subscriptions.push(
    vscode.debug.onDidStartDebugSession((session: vscode.DebugSession) => {
      if (session.type === 'conduit') {
        vscode.window.showInformationMessage(
          `Conduit debug session started: ${session.name}`
        );
      }
    })
  );

  context.subscriptions.push(
    vscode.debug.onDidTerminateDebugSession((session: vscode.DebugSession) => {
      if (session.type === 'conduit') {
        vscode.window.showInformationMessage(
          `Conduit debug session ended: ${session.name}`
        );
      }
    })
  );
}

/**
 * Deactivate debug support
 */
export function deactivateDebug(): void {
  // Cleanup if needed
}

/**
 * Debug configuration interface
 */
export interface ConduitDebugConfiguration extends vscode.DebugConfiguration {
  /** Program to debug (path to compiled binary) */
  program: string;

  /** Arguments to pass to the program */
  args?: string[];

  /** Working directory */
  cwd?: string;

  /** Environment variables */
  env?: { [key: string]: string };

  /** Stop on entry */
  stopOnEntry?: boolean;

  /** Show debug output log */
  showLog?: boolean;

  /** Path to conduit executable */
  conduitPath?: string;

  /** DAP server port (0 = auto) */
  port?: number;

  /** Path to source maps directory */
  sourceMapsPath?: string;
}
