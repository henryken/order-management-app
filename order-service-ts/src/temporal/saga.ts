/**
 * Collects compensations and runs them in reverse order. Like the Java Saga helper with its
 * default options, compensation is sequential and stops at the first compensation that fails.
 */
export class Saga {
  private readonly compensations: Array<() => Promise<void>> = [];

  addCompensation(compensation: () => Promise<void>): void {
    this.compensations.push(compensation);
  }

  async compensate(): Promise<void> {
    for (const compensation of [...this.compensations].reverse()) {
      await compensation();
    }
  }
}
