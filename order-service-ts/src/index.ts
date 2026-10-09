import { Client, Connection } from '@temporalio/client';
import { NativeConnection, Worker } from '@temporalio/worker';
import { createApp } from './api';
import { config } from './config';
import { DownstreamServices } from './infra/downstream';
import { createActivities } from './temporal/activities';
import { ORDER_SERVICE } from './temporal/taskQueues';

async function main() {
  const services = new DownstreamServices(config);
  const workerConnection = await NativeConnection.connect({ address: config.temporalAddress });
  const worker = await Worker.create({
    connection: workerConnection,
    namespace: config.temporalNamespace,
    taskQueue: ORDER_SERVICE,
    workflowsPath: require.resolve('./temporal/workflows'), // For production, use workflowBundle instead
    activities: createActivities(services),
  });

  const connection = await Connection.connect({ address: config.temporalAddress });
  const client = new Client({ connection, namespace: config.temporalNamespace });

  const server = createApp(client, services).listen(config.port, () => {
    console.log(`order-service-ts listening on ${config.port}`);
  });

  // worker.run() resolves once the worker has shut down (it handles SIGINT/SIGTERM itself).
  await worker.run();
  server.close();
  await connection.close();
  await workerConnection.close();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
