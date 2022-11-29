import matplotlib.pyplot as plt
import numpy as np
 
# Theoretical equations
t_N = np.linspace(4, 9.5)
t_orig = 10 / (t_N - 1)
t_coded = 10 / 3 + (t_N * 0)

# Experimental data
e_N = [4, 5, 6, 7, 8, 9]
e_orig = [3.2, 2.27, 1.73, 1.47, 1.2, 1.2]
e_coded = [2.67, 2.67, 2.67, 2.0, 2.13, 2.13]

plt.rcParams.update({'font.size': 14})
plt.axis([4, 9.5, 0, 4])
plt.xlabel('Number of nodes (N)')
plt.ylabel('Throughput (Mbps)')

plt.plot(t_N, t_orig, color="purple", linewidth=2, label='orig (theory)')
plt.plot(t_N, t_coded, color="green", linewidth=2, label='coded (theory)')

plt.scatter(e_N, e_orig, color="purple",
            linewidth=3, marker="o", label='orig (experiment)')

plt.scatter(e_N, e_coded, color="green",
            linewidth=3, marker="o", label='coded (experiment)')

plt.legend()

plt.savefig('throughput.pdf')
