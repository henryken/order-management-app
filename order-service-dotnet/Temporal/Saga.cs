namespace OrderService.Temporal;

/// <summary>
/// Collects compensations and runs them in reverse order. Like the Java Saga helper with its
/// default options, compensation is sequential and stops at the first compensation that fails.
/// </summary>
public sealed class Saga
{
    private readonly List<Func<Task>> compensations = [];

    public void AddCompensation(Func<Task> compensation) => compensations.Add(compensation);

    public async Task CompensateAsync()
    {
        for (var i = compensations.Count - 1; i >= 0; i--)
        {
            await compensations[i]();
        }
    }
}
