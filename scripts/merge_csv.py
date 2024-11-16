import os
import pandas as pd

# used https://tableconvert.com/markdown-to-csv to convert the markdown table to csv

LATENCIES_DIR = '<PATH>'
OUTPUT_FILE = '<PATH>/latency_matrix.csv'

# Set the directory containing the CSV files
directory = LATENCIES_DIR

# Create an empty dataframe to hold the merged data
merged_df = None

# Loop through each file in the directory
for filename in os.listdir(directory):
    if filename.endswith('.csv'):
        # Load the CSV file into a dataframe
        filepath = os.path.join(directory, filename)
        df = pd.read_csv(filepath)

        # Check if 'Source' column exists
        if 'Source' not in df.columns:
            print(f"'Source' column not found in {filename}. Skipping this file.")
            continue

        # Merge the dataframe with the existing merged dataframe
        if merged_df is None:
            merged_df = df  # Initialize with the first dataframe
        else:
            merged_df = pd.merge(merged_df, df, on='Source', how='outer')

# Fill any missing values with 'N/A'
merged_df = merged_df.fillna('N/A')

# Sort the columns alphabetically, with 'Source' column first
column_order = ['Source']
column_order.extend(sorted([col for col in merged_df.columns if col != 'Source']))
merged_df = merged_df[column_order]

# Write the merged dataframe to a new CSV file
merged_df.to_csv(OUTPUT_FILE, index=False)

print("Merging complete! The output file is 'latency_matrix.csv'.")
