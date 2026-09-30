import express from 'express';
import { computeStats } from './stats.js';

export function createApp() {
  const app = express();
  app.use(express.json());

  app.get('/health', (_req, res) => {
    res.json({ status: 'ok' });
  });

  app.post('/api/v1/stats', (req, res) => {
    try {
      const { q, r } = req.body;
      res.json(computeStats(q, r));
    } catch (err) {
      res.status(400).json({ error: err.message });
    }
  });

  return app;
}
