import csv
import random
from datetime import datetime, timedelta

# Test data pools
hcpcs_codes = {
    'K0001': {'desc': 'Standard wheelchair', 'price': 125.50},
    'E0260': {'desc': 'Hospital bed', 'price': 45.75},
    'E0277': {'desc': 'Air pressure pad', 'price': 350.00},
    'A4357': {'desc': 'Catheter', 'price': 12.25},
    'E0651': {'desc': 'Pneumatic pump', 'price': 275.50},
    'L3216': {'desc': 'Orthotic', 'price': 65.25}
}

diagnosis_codes = [
    'E11.621', 'Z89.512', 'S72.001A', 'Z96.641', 'I73.9',
    'M17.12', 'G82.20', 'L89.314', 'Z46.89', 'M54.5'
]

modifiers = ['NU', 'RR', 'LL', 'RA', 'LT', 'RT']
suppliers = ['SUP123', 'SUP456', 'SUP789', 'SUP012', 'SUP345']
providers = ['PRV789', 'PRV456', 'PRV123', 'PRV012', 'PRV345']
transaction_types = ['ORDER', 'CLAIM']

def generate_product_id(hcpcs):
    prefix_map = {
        'K': 'WC',   # Wheelchair
        'E': 'EQ',   # Equipment
        'A': 'SUP',  # Supply
        'L': 'ORT'   # Orthotic
    }
    prefix = prefix_map.get(hcpcs[0], 'DME')
    return f"{prefix}-{hcpcs}-{random.randint(1000, 9999)}"


def generate_test_data(num_records=10000, output_file='test-files/samples/large-transactions.csv'):
    start_date = datetime.now() - timedelta(days=30)

    with open(output_file, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(['transaction_id', 'type', 'patient_id', 'provider_id', 'date',
                        'diagnosis_codes', 'product_id', 'hcpcs_code', 'modifier',
                        'quantity', 'unit_price', 'supplier_id', 'total_amount', 'status'])

        for i in range(num_records):
            # Generate transaction data
            trans_type = random.choice(transaction_types)
            trans_date = start_date + timedelta(
                days=random.randint(0, 29),
                hours=random.randint(0, 23),
                minutes=random.randint(0, 59)
            )

            # Select random HCPCS code and its details
            hcpcs_code = random.choice(list(hcpcs_codes.keys()))
            hcpcs_details = hcpcs_codes[hcpcs_code]

            # Generate other fields
            quantity = random.randint(1, 5)
            unit_price = hcpcs_details['price']
            total_amount = quantity * unit_price

            # Create record
            record = [
                f'TRX{str(i+1).zfill(6)}',                    # transaction_id
                trans_type,                                    # type
                f'PT{random.randint(10000, 99999)}',          # patient_id
                random.choice(providers),                      # provider_id
                trans_date.strftime('%Y-%m-%dT%H:%M:%SZ'),    # date
                ','.join(random.sample(diagnosis_codes, 2)),   # diagnosis_codes
                generate_product_id(hcpcs_code),              # product_id
                hcpcs_code,                                   # hcpcs_code
                random.choice(modifiers),                     # modifier
                quantity,                                     # quantity
                unit_price,                                   # unit_price
                random.choice(suppliers),                     # supplier_id
                total_amount,                                 # total_amount
                'SUBMITTED'                                   # status
            ]

            writer.writerow(record)

if __name__ == '__main__':
    generate_test_data()